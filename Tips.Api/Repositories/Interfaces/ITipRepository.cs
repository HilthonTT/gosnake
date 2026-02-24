using Tips.Api.Models;

namespace Tips.Api.Repositories.Interfaces;

public interface ITipRepository
{
    IReadOnlyList<GameTip> GetAll();
    IReadOnlyList<GameTip> GetByDifficulty(TipDifficulty difficulty);
    IReadOnlyList<GameTip> GetByCategory(TipCategory category);
}
