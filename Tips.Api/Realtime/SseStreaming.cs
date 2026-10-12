using System.Net.ServerSentEvents;
using System.Runtime.CompilerServices;

namespace Tips.Api.Realtime;

internal static class SseStreaming
{
    /// <summary>
    /// Subscribes to <paramref name="broadcaster"/> and yields events (replayed
    /// first, then live) until the client disconnects. Events for which
    /// <paramref name="filter"/> returns false are skipped.
    /// </summary>
    public static async IAsyncEnumerable<SseItem<T>> StreamAsync<T>(
        this SseBroadcaster<T> broadcaster,
        string? lastEventId,
        Func<T, bool>? filter,
        [EnumeratorCancellation] CancellationToken cancellationToken)
    {
        using var subscription = broadcaster.Subscribe(lastEventId);

        await foreach (var item in subscription.Reader.ReadAllAsync(cancellationToken))
        {
            if (filter is null || filter(item.Data))
            {
                yield return item;
            }
        }
    }
}
