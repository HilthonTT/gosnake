using System.ComponentModel.DataAnnotations;

namespace Tips.Api.DTOs.Leaderboard;

public sealed record SubmitScoreRequest(
    [Required, MinLength(1), MaxLength(32)] string PlayerName,
    [Range(0, int.MaxValue)] int Score,
    [Range(1, 10)] int Level,
    [Range(1, int.MaxValue)] int SnakeLength
);
