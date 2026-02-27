namespace Tips.Api.DTOs.Common;

public sealed class CollectionResponse<T> : ICollectionResponse<T>
{
    public List<T> Items { get; init; } = [];
}
