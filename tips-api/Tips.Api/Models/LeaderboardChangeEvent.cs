namespace Tips.Api.Models;

public sealed record LeaderboardChangeEvent(
    LeaderboardChangeType ChangeType,
    LeaderboardEntry Entry);
