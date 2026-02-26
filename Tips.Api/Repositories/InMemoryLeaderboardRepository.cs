using System.Collections.Concurrent;
using Tips.Api.Models;
using Tips.Api.Realtime;
using Tips.Api.Repositories.Interfaces;

namespace Tips.Api.Repositories;

internal sealed class InMemoryLeaderboardRepository(
    LeaderboardBroadcastManager broadcastManager,
    LeaderboardEventBuffer eventBuffer) : ILeaderboardRepository
{
    private readonly ConcurrentDictionary<string, LeaderboardEntry> _entries = new();

    public LeaderboardEntry Add(string playerName, int score, int level, int snakeLength)
    {
        var entry = new LeaderboardEntry(
            EntryId: Guid.NewGuid().ToString("N"),
            PlayerName: playerName,
            Score: score,
            Level: level,
            SnakeLength: snakeLength,
            PlayedAt: DateTimeOffset.UtcNow
        );

        _entries[entry.EntryId] = entry;

        Publish(LeaderboardChangeType.EntryAdded, entry);

        return entry;
    }

    public bool Delete(string entryId)
    {
        if (!_entries.TryRemove(entryId, out var removed))
        {
            return false;
        }

        Publish(LeaderboardChangeType.EntryDeleted, removed);
        return true;
    }

    public IReadOnlyList<LeaderboardEntry> GetAll()
    {
        return [.. _entries.Values.OrderByDescending(e => e.Score).ThenByDescending(e => e.PlayedAt)];
    }

    public IReadOnlyList<LeaderboardEntry> GetByPlayer(string playerName)
    {
        return [.. _entries.Values
            .Where(e => string.Equals(e.PlayerName, playerName, StringComparison.OrdinalIgnoreCase))
            .OrderByDescending(e => e.Score)
            .ThenByDescending(e => e.PlayedAt)];
    }

    public IReadOnlyList<LeaderboardEntry> GetTopN(int count)
    {
        return [.._entries.Values
            .OrderByDescending(e => e.Score)
            .ThenByDescending(e => e.PlayedAt)
            .Take(count)] ;
    }

    private void Publish(LeaderboardChangeType changeType, LeaderboardEntry entry)
    {
        if (broadcastManager.SubscriberCount == 0)
        {
            return;
        }

        var change = new LeaderboardChangeEvent(changeType, entry);
        eventBuffer.Add(change);
        broadcastManager.Broadcast(change);
    }
}
