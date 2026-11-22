using Tips.Api.Models;
using Tips.Api.Repositories.Interfaces;

namespace Tips.Api.Repositories;

internal sealed class InMemoryTipRepository : ITipRepository
{
    private static readonly IReadOnlyList<GameTip> Tips =
    [
        // ── Movement ────────────────────────────────────────────────────────
        new("tip-001", "Plan your path 3–4 moves ahead, not just the next tile.",           TipCategory.Movement,   TipDifficulty.Beginner),
        new("tip-002", "Hug the walls early — it keeps the centre free for longer runs.",   TipCategory.Movement,   TipDifficulty.Beginner),
        new("tip-003", "Never spiral inward without a guaranteed exit route.",              TipCategory.Movement,   TipDifficulty.Intermediate),
        new("tip-004", "Use a Hamiltonian path on small boards to never self-collide.",     TipCategory.Movement,   TipDifficulty.Advanced),
        new("tip-005", "When the board is half-full, start coiling tightly near your tail.",TipCategory.Movement,   TipDifficulty.Advanced),
        new("tip-019", "Move in large loops rather than tight zigzags to preserve space.",  TipCategory.Movement,   TipDifficulty.Beginner),
        new("tip-020", "Avoid cutting the board into two regions — you must live in both.", TipCategory.Movement,   TipDifficulty.Intermediate),
        new("tip-021", "When near a wall, keep your turns parallel to it, not into it.",    TipCategory.Movement,   TipDifficulty.Beginner),
        new("tip-022", "Treat corners as danger zones — always have an escape tile ready.", TipCategory.Movement,   TipDifficulty.Intermediate),
        new("tip-023", "On large boards, divide the grid mentally into quadrants and patrol them in order.", TipCategory.Movement, TipDifficulty.Advanced),
        new("tip-024", "Reversing direction is impossible — never rely on it as an escape plan.", TipCategory.Movement, TipDifficulty.Beginner),
        new("tip-025", "A long snake needs wide turns; start widening your arcs before you grow, not after.", TipCategory.Movement, TipDifficulty.Intermediate),

        // ── Survival ────────────────────────────────────────────────────────
        new("tip-006", "Count free tiles — if they equal your length, you're trapped.",    TipCategory.Survival,   TipDifficulty.Intermediate),
        new("tip-007", "Always keep at least two open directions available.",               TipCategory.Survival,   TipDifficulty.Beginner),
        new("tip-008", "Your tail vacates the square it occupies one tick after you move.", TipCategory.Survival,   TipDifficulty.Intermediate),
        new("tip-009", "Chase your tail when boxed in — it buys time to re-route.",        TipCategory.Survival,   TipDifficulty.Intermediate),
        new("tip-010", "Flood-fill from your head: if reachable tiles < length, reroute.", TipCategory.Survival,   TipDifficulty.Advanced),
        new("tip-026", "If you have one safe move and one risky move, always take the safe one.", TipCategory.Survival, TipDifficulty.Beginner),
        new("tip-027", "The longer your snake, the more dangerous every food pickup becomes — pause and assess.", TipCategory.Survival, TipDifficulty.Intermediate),
        new("tip-028", "A near-miss with your own body is a warning; adjust your route immediately.", TipCategory.Survival, TipDifficulty.Beginner),
        new("tip-029", "Keep a mental map of where your oldest body segments are — they'll free up soonest.", TipCategory.Survival, TipDifficulty.Advanced),
        new("tip-030", "Dying in the centre of the board is more common than near walls — open space creates overconfidence.", TipCategory.Survival, TipDifficulty.Intermediate),
        new("tip-031", "Speed boosts (if available) reduce your reaction window — only use them in open space.", TipCategory.Survival, TipDifficulty.Intermediate),
        new("tip-032", "If two routes reach the food, always take the one that leaves more escape options.",     TipCategory.Survival, TipDifficulty.Advanced),

        // ── Scoring ─────────────────────────────────────────────────────────
        new("tip-011", "Prioritise food that is closest and leaves the most open space.",   TipCategory.Scoring,    TipDifficulty.Intermediate),
        new("tip-012", "Ignore high-risk food early on — consistent survival beats greed.", TipCategory.Scoring,    TipDifficulty.Beginner),
        new("tip-013", "In timed modes, grab food at the edge of your safe region first.",  TipCategory.Scoring,    TipDifficulty.Advanced),
        new("tip-014", "A short safe route to food always beats a long risky shortcut.",    TipCategory.Scoring,    TipDifficulty.Beginner),
        new("tip-033", "High scores come from long unbroken runs, not from risky sprints.", TipCategory.Scoring,    TipDifficulty.Beginner),
        new("tip-034", "If bonus food spawns in a dangerous position, skip it — regular food adds up.", TipCategory.Scoring, TipDifficulty.Intermediate),
        new("tip-035", "Each food pickup makes surviving the next one harder; factor growth into your route before you eat.", TipCategory.Scoring, TipDifficulty.Advanced),
        new("tip-036", "Stringing together pickups in a circuit scores faster than backtracking across the board.", TipCategory.Scoring, TipDifficulty.Intermediate),
        new("tip-037", "Food near walls is often safer to collect than food in the middle — fewer approach angles to worry about.", TipCategory.Scoring, TipDifficulty.Beginner),
        new("tip-038", "In marathon modes your score is mostly determined by how late you die, not how fast you eat.", TipCategory.Scoring, TipDifficulty.Advanced),

        // ── Psychology ──────────────────────────────────────────────────────
        new("tip-015", "Slow down mentally — most mistakes come from reacting too fast.",   TipCategory.Psychology, TipDifficulty.Beginner),
        new("tip-016", "Tunnel vision on the food kills you. Watch the whole board.",       TipCategory.Psychology, TipDifficulty.Beginner),
        new("tip-017", "After a death, replay the last 5 moves in your head before restarting.", TipCategory.Psychology, TipDifficulty.Intermediate),
        new("tip-018", "Set a process goal (e.g. 'no panic turns') not a score goal.",     TipCategory.Psychology, TipDifficulty.Advanced),
        new("tip-039", "Panic is the number one killer in snake. A wrong move made calmly beats a right move made in a rush.", TipCategory.Psychology, TipDifficulty.Beginner),
        new("tip-040", "If you beat your personal best, stop for a moment — chasing it immediately leads to sloppy play.", TipCategory.Psychology, TipDifficulty.Intermediate),
        new("tip-041", "Frustration compounds mistakes. One bad run doesn't predict the next.", TipCategory.Psychology, TipDifficulty.Beginner),
        new("tip-042", "Elite players narrate their plan out loud while practising — it forces deliberate thinking.", TipCategory.Psychology, TipDifficulty.Advanced),
        new("tip-043", "Boredom during safe phases is dangerous — that's when attention drifts and errors creep in.", TipCategory.Psychology, TipDifficulty.Intermediate),
        new("tip-044", "Play the board in front of you, not the score you want. The score follows good decisions.", TipCategory.Psychology, TipDifficulty.Intermediate),
        new("tip-045", "Take short breaks between sessions — motor patterns and spatial reasoning both benefit from rest.", TipCategory.Psychology, TipDifficulty.Advanced),
    ];

    public IReadOnlyList<GameTip> GetAll() => Tips;

    public IReadOnlyList<GameTip> GetByCategory(TipCategory category) =>
        Tips.Where(t => t.Category == category).ToList();

    public IReadOnlyList<GameTip> GetByDifficulty(TipDifficulty difficulty) =>
        Tips.Where(t => t.Difficulty == difficulty).ToList();
}
