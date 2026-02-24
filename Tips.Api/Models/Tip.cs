namespace Tips.Api.Models;

public sealed class Tip
{
    public required TipType Type { get; init; }

    public required string Content { get; init; }
}
