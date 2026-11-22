namespace Tips.Api.Models;

public sealed record LeaderboardEntry(
    string EntryId, 
    string PlayerName, 
    int Score, 
    int Level, 
    int SnakeLength, 
    DateTimeOffset PlayedAt);
