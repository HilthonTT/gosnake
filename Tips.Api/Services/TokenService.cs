using System.Collections.Concurrent;
using System.Security.Cryptography;

namespace Tips.Api.Services;

/// <summary>
/// Issues short-lived, single-use tokens to allow SSE clients to authenticate
/// via query string (browsers cannot send Authorization headers with EventSource).
/// </summary>
internal sealed class TokenService
{
    private readonly ConcurrentDictionary<string, TokenEntry> _tokens = new();

    // Short enough to limit interception risk, long enough to survive a slow redirect.
    private readonly TimeSpan _tokenLifetime = TimeSpan.FromSeconds(10);

    /// <summary>
    /// Generates a secure, unique token for the specified user that can be used for authentication or authorization
    /// purposes.
    /// </summary>
    /// <remarks>This method automatically removes expired tokens from the internal store each time a new
    /// token is generated to maintain optimal performance and prevent unbounded memory usage.</remarks>
    /// <param name="userId">The unique identifier of the user for whom the token is generated. Cannot be null or empty.</param>
    /// <returns>A base64-encoded string representing the generated token.</returns>
    public string GenerateToken(string userId)
    {
        // Purge stale tokens on every issuance to prevent unbounded growth.
        PurgeExpiredTokens();

        string token = Convert.ToBase64String(RandomNumberGenerator.GetBytes(32));
        var entry = new TokenEntry(userId, DateTimeOffset.UtcNow.Add(_tokenLifetime));

        _tokens.TryAdd(token, entry);

        return token;
    }

    /// <summary>
    /// Validates the token and removes it immediately so it cannot be reused.
    /// Returns null if the token is missing, expired, or already consumed.
    /// </summary>
    public string? ValidateAndConsumeToken(string token)
    {
        if (!_tokens.TryRemove(token, out var entry))
        {
            return null;
        }

        if (DateTimeOffset.UtcNow > entry.ExpiresAt)
        {
            return null;
        }

        return entry.UserId;
    }

    private void PurgeExpiredTokens()
    {
        var now = DateTimeOffset.UtcNow;

        foreach (KeyValuePair<string, TokenEntry> kvp in _tokens)
        {
            if (now > kvp.Value.ExpiresAt)
            {
                _tokens.TryRemove(kvp.Key, out _);
            }
        }
    }
}
