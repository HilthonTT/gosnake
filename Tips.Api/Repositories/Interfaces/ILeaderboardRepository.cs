using Tips.Api.Models;

namespace Tips.Api.Repositories.Interfaces;

public interface ILeaderboardRepository
{
    /// <summary>Adds a new entry and returns it with its assigned ID.</summary>
    LeaderboardEntry Add(string playerName, int score, int level, int snakeLength);

    /// <summary>Returns the entry with the given ID, or null if not found.</summary>
    LeaderboardEntry? GetById(string entryId);

    /// <summary>Returns the top <paramref name="count"/> entries by score.</summary>
    IReadOnlyList<LeaderboardEntry> GetTopN(int count);

    /// <summary>Returns all entries for a specific player, best score first.</summary>
    IReadOnlyList<LeaderboardEntry> GetByPlayer(string playerName);

    /// <summary>Removes the entry with the given ID. Returns the removed entry, or null if not found.</summary>
    LeaderboardEntry? Delete(string entryId);
}
