namespace Tips.Api.Services;

public sealed record TokenEntry(string UserId, DateTimeOffset ExpiresAt);

