# Twitter API - Advanced Search

## Endpoint
```
GET /twitter/tweet/advanced_search
```

## Headers
```
X-API-Key: <api-key>
```

## Parameters
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| query | string | ✓ | Поисковый запрос |
| query_type | string | | "Latest" или "Top" (default: Latest) |
| cursor | string | | Пагинация, первая страница = "" |

## Query Examples
```
"AI" OR "Twitter" from:elonmusk since:2021-12-31_23:59:59_UTC
$AAPL OR $TSLA from:username
from:nasa filter:images
#hashtag lang:en min_faves:10
```

## Search Operators
| Operator | Description | Example |
|----------|-------------|---------|
| `from:user` | Твиты от пользователя | `from:elonmusk` |
| `to:user` | Твиты к пользователю | `to:nasa` |
| `$TICKER` | Поиск тикеров | `$AAPL OR $TSLA` |
| `since:YYYY-MM-DD` | С даты | `since:2021-12-31` |
| `until:YYYY-MM-DD` | До даты | `until:2022-01-01` |
| `lang:code` | Язык | `lang:en` |
| `filter:type` | Тип контента | `filter:images` |
| `min_faves:N` | Минимум лайков | `min_faves:10` |
| `-word` | Исключить слово | `-love` |
| `"phrase"` | Точная фраза | `"exact match"` |

## Response
```json
{
  "tweets": [
    {
      "id": "string",
      "text": "string",
      "author": {
        "userName": "string",
        "name": "string"
      },
      "retweetCount": number,
      "likeCount": number,
      "createdAt": "string"
    }
  ],
  "has_next_page": boolean,
  "next_cursor": "string"
}
```

## Notes
- ~20 твитов на страницу
- Используй `cursor` для пагинации
- Короткие запросы работают лучше (1-6 слов)
- Не повторяй одинаковые запросы