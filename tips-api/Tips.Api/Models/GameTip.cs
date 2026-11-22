namespace Tips.Api.Models;

public sealed record GameTip(
    string TipId,
    string Message,
    TipCategory Category,
    TipDifficulty Difficulty
);
