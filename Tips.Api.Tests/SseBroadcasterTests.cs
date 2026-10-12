using System.Diagnostics.Metrics;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Logging.Abstractions;
using Tips.Api.Realtime;

namespace Tips.Api.Tests;

public sealed class SseBroadcasterTests
{
    private static SseBroadcaster<string> CreateBroadcaster(int replayBufferSize = 10)
    {
        var meterFactory = new ServiceCollection().AddMetrics().BuildServiceProvider().GetRequiredService<IMeterFactory>();
        return new SseBroadcaster<string>(
            new SseBroadcasterOptions("Test", "test", ChannelCapacity: 10, ReplayBufferSize: replayBufferSize),
            NullLoggerFactory.Instance,
            meterFactory);
    }

    [Fact]
    public void Publish_AssignsOneIdPerEvent_RegardlessOfSubscriberCount()
    {
        using var broadcaster = CreateBroadcaster();
        using var a = broadcaster.Subscribe(null);
        using var b = broadcaster.Subscribe(null);

        broadcaster.Publish("first");
        broadcaster.Publish("second");

        Assert.True(a.Reader.TryRead(out var a1));
        Assert.True(a.Reader.TryRead(out var a2));
        Assert.True(b.Reader.TryRead(out var b1));
        Assert.True(b.Reader.TryRead(out var b2));

        Assert.Equal(("0", "1"), (a1.EventId, a2.EventId));
        Assert.Equal((a1.EventId, a2.EventId), (b1.EventId, b2.EventId));
        Assert.Equal("test", a1.EventType);
    }

    [Fact]
    public void Subscribe_WithLastEventId_ReplaysOnlyMissedEvents()
    {
        using var broadcaster = CreateBroadcaster();
        broadcaster.Publish("a");
        broadcaster.Publish("b");
        broadcaster.Publish("c");

        using var sub = broadcaster.Subscribe("0");

        Assert.True(sub.Reader.TryRead(out var first));
        Assert.True(sub.Reader.TryRead(out var second));
        Assert.False(sub.Reader.TryRead(out _));
        Assert.Equal(["b", "c"], new[] { first.Data, second.Data });
    }

    [Fact]
    public void ReplayBuffer_IsBounded()
    {
        using var broadcaster = CreateBroadcaster(replayBufferSize: 2);
        for (var i = 0; i < 5; i++)
        {
            broadcaster.Publish($"e{i}");
        }

        using var sub = broadcaster.Subscribe("-1");

        Assert.True(sub.Reader.TryRead(out var first));
        Assert.Equal("e3", first.Data);
    }

    [Fact]
    public void Dispose_RemovesSubscriberAndCompletesReader()
    {
        using var broadcaster = CreateBroadcaster();
        var sub = broadcaster.Subscribe(null);
        Assert.Equal(1, broadcaster.SubscriberCount);

        sub.Dispose();

        Assert.Equal(0, broadcaster.SubscriberCount);
        Assert.True(sub.Reader.Completion.IsCompleted);
    }
}
