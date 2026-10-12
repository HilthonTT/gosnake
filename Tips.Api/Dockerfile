# syntax=docker/dockerfile:1

# Runtime base (also used by Visual Studio's fast-mode container debugging).
FROM mcr.microsoft.com/dotnet/aspnet:10.0 AS base
WORKDIR /app
USER $APP_UID
ENV ASPNETCORE_HTTP_PORTS=8080
EXPOSE 8080

FROM mcr.microsoft.com/dotnet/sdk:10.0 AS build
ARG BUILD_CONFIGURATION=Release
WORKDIR /src
COPY ["Tips.Api.csproj", "."]
RUN dotnet restore "./Tips.Api.csproj"
COPY . .
RUN dotnet publish "./Tips.Api.csproj" -c $BUILD_CONFIGURATION -o /app/publish --no-restore /p:UseAppHost=false

FROM base AS final
COPY --from=build /app/publish .
ENTRYPOINT ["dotnet", "Tips.Api.dll"]
