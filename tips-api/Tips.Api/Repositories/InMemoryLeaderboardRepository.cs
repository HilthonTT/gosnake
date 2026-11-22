using System.Collections.Concurrent;
using Tips.Api.Models;
using Tips.Api.Repositories.Interfaces;

namespace Tips.Api.Repositories;

internal sealed class InMemoryLeaderboardRepository(TimeProvider timeProvider) : ILeaderboardRepository
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
            PlayedAt: timeProvider.GetUtcNow());

        _entries[entry.EntryId] = entry;
        return entry;
    }

    public LeaderboardEntry? GetById(string entryId) =>
        _entries.TryGetValue(entryId, out var entry) ? entry : null;

    public LeaderboardEntry? Delete(string entryId) =>
        _entries.TryRemove(entryId, out var removed) ? removed : null;

    public IReadOnlyList<LeaderboardEntry> GetByPlayer(string playerName) =>
        [.. Ranked(_entries.Values.Where(e =>
            string.Equals(e.PlayerName, playerName, StringComparison.OrdinalIgnoreCase)))];

    public IReadOnlyList<LeaderboardEntry> GetTopN(int count) =>
        [.. Ranked(_entries.Values).Take(count)];

    private static IEnumerable<LeaderboardEntry> Ranked(IEnumerable<LeaderboardEntry> entries) =>
        entries.OrderByDescending(e => e.Score).ThenByDescending(e => e.PlayedAt);
}
