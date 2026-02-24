using System.Net.ServerSentEvents;
using Tips.Api.Models;

namespace Tips.Api.Realtime;

/// <summary>
/// Stores recently broadcast tips so reconnecting clients can replay
/// anything they missed via the Last-Event-ID header.
///
/// Unlike the Orders demo this is a single global buffer — all clients
/// see the same tip stream so there is no need for per-user partitioning.
/// </summary>
internal sealed class TipEventBuffer(int maxBufferSize = 50)
{
    private readonly LinkedList<SseItem<GameTip>> _buffer = new();
    private long _nextEventId = 0;
    private readonly Lock _lock = new();

    /// <summary>
    /// Adds a tip to the buffer and assigns it a monotonically increasing event ID.
    /// Thread-safe: the ID assignment and enqueue are atomic under the lock.
    /// </summary>
    public SseItem<GameTip> Add(GameTip tip)
    {
        lock (_lock)
        {
            var eventId = _nextEventId++;
            var item = new SseItem<GameTip>(tip) { EventId = eventId.ToString() };

            _buffer.AddLast(item);
            while (_buffer.Count > maxBufferSize)
            {
                _buffer.RemoveFirst();
            }

            return item;
        }
    }

    /// <summary>
    /// Returns all buffered tips with an event ID greater than lastEventId,
    /// in chronological order. Used to replay missed events on reconnect.
    /// </summary>
    public IReadOnlyList<SseItem<GameTip>> GetEventsAfter(string? lastEventId)
    {
        if (string.IsNullOrEmpty(lastEventId) || !long.TryParse(lastEventId, out var lastId))
        {
            return [];
        }

        lock (_lock)
        {
            return _buffer
                .Where(item => long.TryParse(item.EventId, out var id) && id > lastId)
                .ToList();
        }
    }

    public long CurrentEventId
    {
        get { lock (_lock) return _nextEventId - 1; }
    }
}
