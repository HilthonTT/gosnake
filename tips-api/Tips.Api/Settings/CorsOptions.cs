namespace Tips.Api.Settings;

public sealed class CorsOptions
{
    public const string PolicyName = "SnakeGameTipsCorsPolicy";
    public const string SectionName = "Cors";
    public required string[] AllowedOrigins { get; init; }
}
