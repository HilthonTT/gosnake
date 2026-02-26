using System.Net.ServerSentEvents;
using Tips.Api.Models;

namespace Tips.Api.Realtime;

internal sealed class LeaderboardEventBuffer(int maxBufferSize = 100)
{
    private readonly LinkedList<SseItem<LeaderboardChangeEvent>> _buffer = new();
    private long _nextEventId = 0;
    private readonly Lock _lock = new();

    /// <summary>
    /// Wraps <paramref name="change"/> in an <see cref="SseItem{T}"/>, assigns it
    /// a monotonically increasing event ID, appends it to the buffer, and returns
    /// the item ready for streaming.
    /// </summary>
    public SseItem<LeaderboardChangeEvent> Add(LeaderboardChangeEvent change)
    {
        lock (_lock)
        {
            var eventId = _nextEventId++;

            var item = new SseItem<LeaderboardChangeEvent>(change)
            {
                EventId = eventId.ToString(),
            };

            _buffer.AddLast(item);

            // Avoids the buffer from growing too larger
            while (_buffer.Count > maxBufferSize)
            {
                _buffer.RemoveFirst();
            }

            return item;
        }
    }

    /// <summary>
    /// Returns all buffered items with an event ID strictly greater than
    /// <paramref name="lastEventId"/>, in chronological order.
    /// Returns an empty list when <paramref name="lastEventId"/> is null/unparseable.
    /// </summary>
    public IReadOnlyList<SseItem<LeaderboardChangeEvent>> GetEventsAfter(string? lastEventId)
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
