using Tips.Api.Models;

namespace Tips.Api.Repositories.Interfaces;

public interface ILeaderboardRepository
{
    /// <summary>Adds a new entry and returns it with its assigned ID.</summary>
    LeaderboardEntry Add(string playerName, int score, int level, int snakeLength);

    /// <summary>Returns all entries ordered by score descending.</summary>
    IReadOnlyList<LeaderboardEntry> GetAll();

    /// <summary>Returns the top <paramref name="count"/> entries by score.</summary>
    IReadOnlyList<LeaderboardEntry> GetTopN(int count);

    /// <summary>Returns all entries for a specific player, best score first.</summary>
    IReadOnlyList<LeaderboardEntry> GetByPlayer(string playerName);

    /// <summary>Removes the entry with the given ID. Returns false if not found.</summary>
    bool Delete(string entryId);
}
