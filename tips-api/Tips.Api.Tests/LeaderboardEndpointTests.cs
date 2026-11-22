using System.Net;
using System.Net.Http.Json;
using System.Text.Json;
using System.Text.Json.Serialization;
using Microsoft.AspNetCore.Mvc.Testing;
using Tips.Api.DTOs.Common;
using Tips.Api.Models;

namespace Tips.Api.Tests;

public sealed class LeaderboardEndpointTests(WebApplicationFactory<Program> factory)
    : IClassFixture<WebApplicationFactory<Program>>
{
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web)
    {
        Converters = { new JsonStringEnumConverter() },
    };

    private readonly HttpClient _client = factory.CreateClient();

    [Fact]
    public async Task SubmitScore_ThenGet_ReturnsEntryRankedByScore()
    {
        var name = $"player-{Guid.NewGuid():N}"[..20];

        var low = await _client.PostAsJsonAsync("/api/v1/leaderboard", new { playerName = name, score = 10, level = 1, snakeLength = 2 });
        var high = await _client.PostAsJsonAsync("/api/v1/leaderboard", new { playerName = name, score = 90, level = 2, snakeLength = 10 });

        Assert.Equal(HttpStatusCode.Created, low.StatusCode);
        Assert.Equal(HttpStatusCode.Created, high.StatusCode);

        var created = await high.Content.ReadFromJsonAsync<LeaderboardEntry>();
        Assert.NotNull(created);
        Assert.Equal($"/api/v1/leaderboard/{created.EntryId}", high.Headers.Location?.OriginalString);

        var byPlayer = await _client.GetFromJsonAsync<CollectionResponse<LeaderboardEntry>>($"/api/v1/leaderboard/player/{name}");
        Assert.NotNull(byPlayer);
        Assert.Equal([90, 10], byPlayer.Items.Select(e => e.Score));

        var byId = await _client.GetFromJsonAsync<LeaderboardEntry>($"/api/v1/leaderboard/{created.EntryId}");
        Assert.Equal(created, byId);
    }

    [Fact]
    public async Task SubmitScore_WithInvalidPayload_Returns400()
    {
        var response = await _client.PostAsJsonAsync("/api/v1/leaderboard", new { playerName = "", score = -1, level = 99, snakeLength = 0 });

        Assert.Equal(HttpStatusCode.BadRequest, response.StatusCode);
    }

    [Fact]
    public async Task DeleteEntry_UnknownId_Returns404()
    {
        var response = await _client.DeleteAsync("/api/v1/leaderboard/does-not-exist");

        Assert.Equal(HttpStatusCode.NotFound, response.StatusCode);
    }

    [Fact]
    public async Task GetTips_FiltersByDifficulty()
    {
        var tips = await _client.GetFromJsonAsync<CollectionResponse<GameTip>>("/api/v1/tips?difficulty=Advanced", JsonOptions);

        Assert.NotNull(tips);
        Assert.NotEmpty(tips.Items);
        Assert.All(tips.Items, t => Assert.Equal(TipDifficulty.Advanced, t.Difficulty));
    }

    [Fact]
    public async Task Health_ReturnsOk()
    {
        var response = await _client.GetAsync("/health");

        Assert.Equal(HttpStatusCode.OK, response.StatusCode);
    }

    [Fact]
    public async Task LeaderboardStream_SendsEnumsAsNames()
    {
        using var cts = new CancellationTokenSource(TimeSpan.FromSeconds(10));
        await _client.PostAsJsonAsync("/api/v1/leaderboard", new { playerName = "sse", score = 1, level = 1, snakeLength = 1 }, cts.Token);

        // Replay from the beginning so the event is delivered without racing the POST.
        using var request = new HttpRequestMessage(HttpMethod.Get, "/api/v1/leaderboard/realtime");
        request.Headers.Add("Last-Event-ID", "-1");
        using var response = await _client.SendAsync(request, HttpCompletionOption.ResponseHeadersRead, cts.Token);
        using var reader = new StreamReader(await response.Content.ReadAsStreamAsync(cts.Token));

        string? line;
        do
        {
            line = await reader.ReadLineAsync(cts.Token);
        }
        while (line is not null && !line.StartsWith("data:", StringComparison.Ordinal));

        Assert.NotNull(line);
        Assert.Contains("\"changeType\":\"EntryAdded\"", line);
    }

    [Fact]
    public async Task GetTips_SerializesEnumsAsNames()
    {
        var json = await _client.GetStringAsync("/api/v1/tips?category=Scoring");

        Assert.Contains("\"category\":\"Scoring\"", json);
    }
}
