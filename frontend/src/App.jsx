import { useMemo, useState } from "react";
import {
  Alert,
  AppBar,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  Container,
  Divider,
  MenuItem,
  Paper,
  Stack,
  TextField,
  Toolbar,
  Typography,
} from "@mui/material";
import SportsEsportsRoundedIcon from "@mui/icons-material/SportsEsportsRounded";
import QueryStatsRoundedIcon from "@mui/icons-material/QueryStatsRounded";
import EmojiEventsRoundedIcon from "@mui/icons-material/EmojiEventsRounded";

const defaultGameBaseUrl = import.meta.env.VITE_GAME_SERVICE_URL || "http://localhost:8080";
const defaultRatingBaseUrl = import.meta.env.VITE_RATING_SERVICE_URL || "http://localhost:8081";

const initialState = {
  gameBaseUrl: defaultGameBaseUrl,
  ratingBaseUrl: defaultRatingBaseUrl,
  gameWhitePlayerId: "player-1",
  gameBlackPlayerId: "player-2",
  gameId: "",
  resignGameId: "",
  resignPlayerColor: "black",
  ratingPlayerIdCreate: "player-1",
  ratingInitialValue: "1200",
  ratingPlayerIdGet: "player-1",
  leaderboardLimit: "10",
};

function SectionCard({ title, subtitle, icon, children }) {
  return (
    <Card
      elevation={0}
      sx={{
        height: "100%",
        border: "1px solid",
        borderColor: "divider",
        background: "linear-gradient(180deg, #fffdf9 0%, #faf5ee 100%)",
      }}
    >
      <CardContent sx={{ p: 3 }}>
        <Stack direction="row" spacing={1.5} alignItems="center" sx={{ mb: 1 }}>
          {icon}
          <Typography variant="h5">{title}</Typography>
        </Stack>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
          {subtitle}
        </Typography>
        {children}
      </CardContent>
    </Card>
  );
}

export default function App() {
  const [form, setForm] = useState(initialState);
  const [demoLoading, setDemoLoading] = useState(false);
  const [response, setResponse] = useState({
    title: "Ответ сервера",
    body: "Здесь появится JSON-ответ.",
    type: "info",
  });

  const prettyResponse = useMemo(() => {
    if (typeof response.body === "string") {
      return response.body;
    }

    return JSON.stringify(response.body, null, 2);
  }, [response]);

  function updateField(key, value) {
    setForm((current) => ({
      ...current,
      [key]: value,
    }));
  }

  async function runDemoScenario() {
    const stamp = Date.now();
    const whitePlayerId = `demo-white-${stamp}`;
    const blackPlayerId = `demo-black-${stamp}`;

    setDemoLoading(true);

    try {
      const createGameResponse = await fetch(`${form.gameBaseUrl}/api/v1/games`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          whitePlayerId,
          blackPlayerId,
        }),
      });
      const createdGame = await createGameResponse.json();

      if (!createGameResponse.ok || !createdGame.id) {
        throw new Error("Не удалось создать демонстрационную партию");
      }

      const resignResponse = await fetch(`${form.gameBaseUrl}/api/v1/games/${createdGame.id}/resign`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          playerColor: "black",
        }),
      });
      const resignedGame = await resignResponse.json();

      if (!resignResponse.ok) {
        throw new Error("Не удалось завершить демонстрационную партию");
      }

      const ratingResponse = await fetch(`${form.ratingBaseUrl}/api/v1/ratings/players/${whitePlayerId}`);
      const ratingProfile = await ratingResponse.json();

      if (!ratingResponse.ok) {
        throw new Error("Не удалось получить рейтинг после демо-сценария");
      }

      setForm((current) => ({
        ...current,
        gameWhitePlayerId: whitePlayerId,
        gameBlackPlayerId: blackPlayerId,
        gameId: createdGame.id,
        resignGameId: createdGame.id,
        ratingPlayerIdGet: whitePlayerId,
        ratingPlayerIdCreate: whitePlayerId,
      }));

      setResponse({
        title: "Демо-сценарий завершен",
        type: "success",
        body: {
          description: "Создали партию, завершили ее сдачей черных и получили обновленный рейтинг белых.",
          createdGame,
          resignedGame,
          whiteRatingProfile: ratingProfile,
        },
      });
    } catch (error) {
      setResponse({
        title: "Демо-сценарий завершился ошибкой",
        type: "error",
        body: {
          message: error.message,
        },
      });
    } finally {
      setDemoLoading(false);
    }
  }

  async function sendJsonRequest(title, url, options = {}) {
    try {
      const result = await fetch(url, options);
      const text = await result.text();

      let payload;
      try {
        payload = text ? JSON.parse(text) : { ok: true };
      } catch {
        payload = { raw: text };
      }

      setResponse({
        title: `${title} [${result.status}]`,
        body: payload,
        type: result.ok ? "success" : "warning",
      });
    } catch (error) {
      setResponse({
        title: `${title} [network error]`,
        body: { message: error.message },
        type: "error",
      });
    }
  }

  return (
    <Box sx={{ minHeight: "100vh", background: "linear-gradient(180deg, #f6f1e8 0%, #efe5d5 100%)" }}>
      <AppBar
        position="static"
        elevation={0}
        sx={{
          background: "linear-gradient(90deg, #0d5c63 0%, #155d7a 100%)",
        }}
      >
        <Toolbar sx={{ minHeight: 76 }}>
          <Stack direction="row" spacing={1.5} alignItems="center">
            <SportsEsportsRoundedIcon />
            <Box>
              <Typography variant="h6">Chess Platform</Typography>
              <Typography variant="body2" sx={{ opacity: 0.86 }}>
                Простой React + MUI интерфейс для двух микросервисов
              </Typography>
            </Box>
          </Stack>
        </Toolbar>
      </AppBar>

      <Container maxWidth="xl" sx={{ py: 4 }}>
        <Paper
          elevation={0}
          sx={{
            p: 3,
            mb: 3,
            border: "1px solid",
            borderColor: "divider",
            background: "rgba(255,255,255,0.72)",
            backdropFilter: "blur(12px)",
          }}
        >
          <Stack spacing={2}>
            <Box>
              <Typography variant="h3" gutterBottom>
                Панель управления микросервисами
              </Typography>
              <Typography variant="body1" color="text.secondary">
                Здесь можно проверить состояние сервисов, создать партию, завершить игру и
                сразу увидеть обновление рейтинга.
              </Typography>
            </Box>

            <Stack direction={{ xs: "column", md: "row" }} spacing={2}>
              <TextField
                fullWidth
                label="Game Service URL"
                value={form.gameBaseUrl}
                onChange={(event) => updateField("gameBaseUrl", event.target.value)}
              />
              <TextField
                fullWidth
                label="Rating Service URL"
                value={form.ratingBaseUrl}
                onChange={(event) => updateField("ratingBaseUrl", event.target.value)}
              />
            </Stack>

            <Stack direction="row" spacing={1} useFlexGap flexWrap="wrap">
              <Chip label={`Game Service: ${defaultGameBaseUrl}`} color="primary" variant="outlined" />
              <Chip label={`Rating Service: ${defaultRatingBaseUrl}`} color="secondary" variant="outlined" />
              <Chip label="Frontend: localhost:3000" variant="outlined" />
            </Stack>
          </Stack>
        </Paper>

        <Paper
          elevation={0}
          sx={{
            p: 3,
            mb: 3,
            border: "1px solid",
            borderColor: "divider",
            background: "linear-gradient(135deg, #fff8ee 0%, #fff3df 100%)",
          }}
        >
          <Stack spacing={2}>
            <Box>
              <Typography variant="h4" gutterBottom>
                Демо-сценарий
              </Typography>
              <Typography variant="body1" color="text.secondary">
                Одна кнопка для быстрого показа связки двух микросервисов:
                создаем партию, завершаем ее сдачей и сразу получаем обновленный рейтинг.
              </Typography>
            </Box>

            <Stack direction={{ xs: "column", sm: "row" }} spacing={2} alignItems={{ xs: "stretch", sm: "center" }}>
              <Button
                size="large"
                variant="contained"
                color="secondary"
                disabled={demoLoading}
                onClick={runDemoScenario}
              >
                {demoLoading ? "Выполняется..." : "Запустить демо"}
              </Button>
              <Chip
                label="Используются новые demo-player id, чтобы сценарий был чистым"
                variant="outlined"
              />
            </Stack>
          </Stack>
        </Paper>

        <Box
          sx={{
            display: "grid",
            gap: 3,
            gridTemplateColumns: {
              xs: "1fr",
              lg: "1fr 1fr",
            },
          }}
        >
          <Box>
            <SectionCard
              title="Сервис партий"
              subtitle="Создание партии, просмотр состояния и завершение игры по сдаче."
              icon={<QueryStatsRoundedIcon color="primary" />}
            >
              <Stack spacing={2.5}>
                <Stack direction={{ xs: "column", sm: "row" }} spacing={1.5}>
                  <Button variant="contained" onClick={() => sendJsonRequest("Game /health", `${form.gameBaseUrl}/health`)}>
                    Health
                  </Button>
                  <Button variant="outlined" onClick={() => sendJsonRequest("Game /stats", `${form.gameBaseUrl}/stats`)}>
                    Stats
                  </Button>
                </Stack>

                <Divider />

                <Typography variant="h6">Создать партию</Typography>
                <Stack direction={{ xs: "column", sm: "row" }} spacing={2}>
                  <TextField
                    fullWidth
                    label="White Player ID"
                    value={form.gameWhitePlayerId}
                    onChange={(event) => updateField("gameWhitePlayerId", event.target.value)}
                  />
                  <TextField
                    fullWidth
                    label="Black Player ID"
                    value={form.gameBlackPlayerId}
                    onChange={(event) => updateField("gameBlackPlayerId", event.target.value)}
                  />
                </Stack>
                <Button
                  variant="contained"
                  onClick={() =>
                    sendJsonRequest("Create Game", `${form.gameBaseUrl}/api/v1/games`, {
                      method: "POST",
                      headers: { "Content-Type": "application/json" },
                      body: JSON.stringify({
                        whitePlayerId: form.gameWhitePlayerId.trim(),
                        blackPlayerId: form.gameBlackPlayerId.trim(),
                      }),
                    })
                  }
                >
                  Создать партию
                </Button>

                <Divider />

                <Typography variant="h6">Получить партию</Typography>
                <TextField
                  fullWidth
                  label="Game ID"
                  value={form.gameId}
                  onChange={(event) => updateField("gameId", event.target.value)}
                />
                <Button
                  variant="outlined"
                  onClick={() =>
                    sendJsonRequest("Get Game", `${form.gameBaseUrl}/api/v1/games/${form.gameId.trim()}`)
                  }
                >
                  Получить партию
                </Button>

                <Divider />

                <Typography variant="h6">Сдаться</Typography>
                <Stack direction={{ xs: "column", sm: "row" }} spacing={2}>
                  <TextField
                    fullWidth
                    label="Game ID"
                    value={form.resignGameId}
                    onChange={(event) => updateField("resignGameId", event.target.value)}
                  />
                  <TextField
                    select
                    fullWidth
                    label="Player Color"
                    value={form.resignPlayerColor}
                    onChange={(event) => updateField("resignPlayerColor", event.target.value)}
                  >
                    <MenuItem value="white">white</MenuItem>
                    <MenuItem value="black">black</MenuItem>
                  </TextField>
                </Stack>
                <Button
                  color="secondary"
                  variant="contained"
                  onClick={() =>
                    sendJsonRequest(
                      "Resign Game",
                      `${form.gameBaseUrl}/api/v1/games/${form.resignGameId.trim()}/resign`,
                      {
                        method: "POST",
                        headers: { "Content-Type": "application/json" },
                        body: JSON.stringify({
                          playerColor: form.resignPlayerColor,
                        }),
                      },
                    )
                  }
                >
                  Завершить партию по сдаче
                </Button>
              </Stack>
            </SectionCard>
          </Box>

          <Box>
            <SectionCard
              title="Сервис рейтингов"
              subtitle="Профили игроков, Elo-рейтинг и таблица лидеров."
              icon={<EmojiEventsRoundedIcon color="secondary" />}
            >
              <Stack spacing={2.5}>
                <Stack direction={{ xs: "column", sm: "row" }} spacing={1.5}>
                  <Button variant="contained" onClick={() => sendJsonRequest("Rating /health", `${form.ratingBaseUrl}/health`)}>
                    Health
                  </Button>
                  <Button variant="outlined" onClick={() => sendJsonRequest("Rating /stats", `${form.ratingBaseUrl}/stats`)}>
                    Stats
                  </Button>
                </Stack>

                <Divider />

                <Typography variant="h6">Создать профиль</Typography>
                <Stack direction={{ xs: "column", sm: "row" }} spacing={2}>
                  <TextField
                    fullWidth
                    label="Player ID"
                    value={form.ratingPlayerIdCreate}
                    onChange={(event) => updateField("ratingPlayerIdCreate", event.target.value)}
                  />
                  <TextField
                    fullWidth
                    label="Initial Rating"
                    value={form.ratingInitialValue}
                    onChange={(event) => updateField("ratingInitialValue", event.target.value)}
                  />
                </Stack>
                <Button
                  variant="contained"
                  onClick={() =>
                    sendJsonRequest("Create Rating Profile", `${form.ratingBaseUrl}/api/v1/ratings/players`, {
                      method: "POST",
                      headers: { "Content-Type": "application/json" },
                      body: JSON.stringify({
                        playerId: form.ratingPlayerIdCreate.trim(),
                        initialRating: Number(form.ratingInitialValue || "0"),
                      }),
                    })
                  }
                >
                  Создать профиль
                </Button>

                <Divider />

                <Typography variant="h6">Получить профиль игрока</Typography>
                <TextField
                  fullWidth
                  label="Player ID"
                  value={form.ratingPlayerIdGet}
                  onChange={(event) => updateField("ratingPlayerIdGet", event.target.value)}
                />
                <Button
                  variant="outlined"
                  onClick={() =>
                    sendJsonRequest(
                      "Get Rating Profile",
                      `${form.ratingBaseUrl}/api/v1/ratings/players/${form.ratingPlayerIdGet.trim()}`,
                    )
                  }
                >
                  Получить профиль
                </Button>

                <Divider />

                <Typography variant="h6">Leaderboard</Typography>
                <TextField
                  fullWidth
                  label="Limit"
                  value={form.leaderboardLimit}
                  onChange={(event) => updateField("leaderboardLimit", event.target.value)}
                />
                <Button
                  color="secondary"
                  variant="outlined"
                  onClick={() =>
                    sendJsonRequest(
                      "Leaderboard",
                      `${form.ratingBaseUrl}/api/v1/ratings/leaderboard?limit=${form.leaderboardLimit.trim()}`,
                    )
                  }
                >
                  Показать лидерборд
                </Button>
              </Stack>
            </SectionCard>
          </Box>

          <Paper
            elevation={0}
            sx={{
              p: 3,
              border: "1px solid",
              borderColor: "divider",
              background: "#141414",
              color: "#f5f5f5",
            }}
          >
            <Stack spacing={2}>
              <Alert severity={response.type} variant="filled">
                {response.title}
              </Alert>
              <Box
                component="pre"
                sx={{
                  m: 0,
                  p: 2,
                  overflow: "auto",
                  borderRadius: 2,
                  backgroundColor: "#1f1f1f",
                  color: "#f5f5f5",
                  fontSize: 14,
                  lineHeight: 1.5,
                  whiteSpace: "pre-wrap",
                  wordBreak: "break-word",
                }}
              >
                {prettyResponse}
              </Box>
            </Stack>
          </Paper>
        </Box>
      </Container>
    </Box>
  );
}
