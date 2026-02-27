using FluentValidation;

namespace Tips.Api.DTOs.Leaderboard;

internal sealed class SubmitScoreRequestValidator : AbstractValidator<SubmitScoreRequest>
{
    public SubmitScoreRequestValidator()
    {
        RuleFor(x => x.Score)
            .GreaterThanOrEqualTo(0)
            .WithMessage("Score must be non-negative");

        RuleFor(x => x.PlayerName)
            .NotEmpty()
            .WithMessage("Player name is required.");

        RuleFor(x => x.Level)
            .InclusiveBetween(1, 10)
            .WithMessage("Level must be between 1 and 10.");

        RuleFor(x => x.SnakeLength)
            .GreaterThanOrEqualTo(1).WithMessage("Snake length must be at least 1.");
    }
}
