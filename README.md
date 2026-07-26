# SmartlyQ Go SDK

The official Go SDK for the [SmartlyQ API](https://docs.smartlyq.com) - social posting and scheduling, AI content generation (articles, images, video, audio, presentations), SEO research, CRM, chatbots, and more, from one API key.

- **Zero dependencies** - built on `net/http` and `encoding/json` only.
- **Batteries included** - automatic retries with backoff, idempotency keys, request timeouts, typed errors.
- **Context aware** - every method takes a `context.Context` for cancellation and deadlines.

## Installation

```bash
go get github.com/SmartlyQ/smartlyq-go
```

Requires Go 1.22+.

## Quickstart

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	smartlyq "github.com/SmartlyQ/smartlyq-go"
)

func main() {
	client := smartlyq.NewClient(os.Getenv("SMARTLYQ_API_KEY"))
	ctx := context.Background()

	// Who am I?
	me, err := client.Account.GetMe(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(me.Data))

	// Generate an image with AI
	image, err := client.Images.Generate(ctx, map[string]any{
		"prompt": "A minimalist product shot of a smart speaker",
	}, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(image.Data))

	// Publish a social post
	post, err := client.Social.CreatePost(ctx, map[string]any{
		"text":        "Hello from the SmartlyQ SDK!",
		"account_ids": []string{"acc_123"},
	}, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(post.Data))
}
```

Get an API key from your [Developer Dashboard](https://app.smartlyq.com). Keys look like `sqk_live_...` (production) or `sqk_test_...` (sandbox - free simulated responses, no charges).

## Configuration

```go
client := smartlyq.NewClient(
	"sqk_live_xxxxxxxxxxxx", // or leave empty to use SMARTLYQ_API_KEY
	smartlyq.WithTimeout(60*time.Second),   // per-request timeout
	smartlyq.WithMaxRetries(2),             // automatic retries on 429/5xx
	smartlyq.WithBaseURL("https://api.smartlyq.com/v1"),
	smartlyq.WithHTTPClient(&http.Client{}),
)
```

Per-request options are accepted as the last argument of every method (pass `nil` for defaults):

```go
post, err := client.Social.CreatePost(ctx, body, &smartlyq.RequestOptions{
	IdempotencyKey: "my-unique-key", // safe retries for writes
	ProfileID:      "prof_123",      // act on behalf of a managed Profile
})
```

## Responses

Every method returns a `*smartlyq.Envelope`. The `Data` field is raw JSON so you can unmarshal it into your own types:

```go
resp, err := client.Jobs.Get(ctx, "job_123", nil)
if err != nil {
	log.Fatal(err)
}

var job struct {
	Status string          `json:"status"`
	Result json.RawMessage `json:"result"`
}
if err := resp.UnmarshalData(&job); err != nil {
	log.Fatal(err)
}
```

## Async jobs

Generation endpoints (articles, images, videos, audio) are asynchronous: they return a job. Poll it until it completes:

```go
resp, _ := client.Videos.Generate(ctx, map[string]any{
	"prompt": "A 5s product teaser",
	"model":  "standard",
}, nil)

var gen struct {
	JobID string `json:"job_id"`
}
_ = resp.UnmarshalData(&gen)

for {
	jobResp, err := client.Jobs.Get(ctx, gen.JobID, nil)
	if err != nil {
		log.Fatal(err)
	}
	var job struct {
		Status string `json:"status"`
	}
	_ = jobResp.UnmarshalData(&job)
	if job.Status != "processing" && job.Status != "queued" {
		break
	}
	time.Sleep(3 * time.Second)
}
```

## Error handling

Every non-2xx response returns a typed `*smartlyq.APIError`:

```go
_, err := client.Articles.Generate(ctx, map[string]any{"topic": "AI trends"}, nil)
if err != nil {
	var apiErr *smartlyq.APIError
	if errors.As(err, &apiErr) {
		fmt.Println(apiErr.StatusCode, apiErr.Code, apiErr.Message, apiErr.RequestID)
	}
}
```

## API Reference

All methods below are available on the client. Every method also accepts a trailing `opts *smartlyq.RequestOptions` argument (shown here as omitted - pass `nil` for defaults). Full request/response documentation lives at [docs.smartlyq.com](https://docs.smartlyq.com).

<!-- BEGIN GENERATED REFERENCE -->

### Account

| Method | Endpoint | Description |
| --- | --- | --- |
| `client.Account.GetMe(ctx)` | `GET /me` | Get current user profile |
| `client.Account.GetMeUsage(ctx, query)` | `GET /me/usage` | Get usage summary |
| `client.Account.GetMeBalance(ctx)` | `GET /me/balance` | Get wallet balance |

### AI Captain

| Method | Endpoint | Description |
| --- | --- | --- |
| `client.Captain.SendMessage(ctx, body)` | `POST /captain/messages` | Send AI Captain message |
| `client.Captain.ListConversations(ctx, query)` | `GET /captain/conversations` | List AI Captain conversations |
| `client.Captain.GetConversation(ctx, conversationId)` | `GET /captain/conversations/{conversation_id}` | Get AI Captain conversation |

### Analytics

| Method | Endpoint | Description |
| --- | --- | --- |
| `client.Analytics.GetOverview(ctx, query)` | `GET /analytics/overview` | Get analytics overview |
| `client.Analytics.GetPosts(ctx, query)` | `GET /analytics/posts` | Get post analytics |
| `client.Analytics.GetAccount(ctx, accountId, query)` | `GET /analytics/accounts/{account_id}` | Get account analytics |
| `client.Analytics.DailyMetrics(ctx, query)` | `GET /analytics/daily-metrics` | Daily metrics |
| `client.Analytics.BestTime(ctx, query)` | `GET /analytics/best-time` | Best time to post |
| `client.Analytics.ContentDecay(ctx, query)` | `GET /analytics/content-decay` | Content decay |
| `client.Analytics.PostingFrequency(ctx, query)` | `GET /analytics/posting-frequency` | Posting frequency vs engagement |
| `client.Analytics.PostTimeline(ctx, postId)` | `GET /analytics/posts/{post_id}/timeline` | Post metric timeline |

### Articles

| Method | Endpoint | Description |
| --- | --- | --- |
| `client.Articles.Generate(ctx, body)` | `POST /articles/generate` | Generate article |
| `client.Articles.List(ctx, query)` | `GET /articles` | List articles |
| `client.Articles.Get(ctx, articleId)` | `GET /articles/{article_id}` | Get article |
| `client.Articles.Delete(ctx, articleId)` | `DELETE /articles/{article_id}` | Delete article |

### Audio

| Method | Endpoint | Description |
| --- | --- | --- |
| `client.Audio.TextToSpeech(ctx, body)` | `POST /audio/text-to-speech` | Text to speech |
| `client.Audio.SpeechToText(ctx, body)` | `POST /audio/speech-to-text` | Speech to text |
| `client.Audio.Get(ctx, audioId)` | `GET /audio/{audio_id}` | Get audio |

### Chatbot

| Method | Endpoint | Description |
| --- | --- | --- |
| `client.Chatbots.List(ctx, query)` | `GET /chatbots` | List chatbots |
| `client.Chatbots.Create(ctx, body)` | `POST /chatbots` | Create chatbot |
| `client.Chatbots.Get(ctx, id)` | `GET /chatbots/{id}` | Get chatbot |
| `client.Chatbots.Update(ctx, id, body)` | `PATCH /chatbots/{id}` | Update chatbot |
| `client.Chatbots.Delete(ctx, id)` | `DELETE /chatbots/{id}` | Delete chatbot |
| `client.Chatbots.Train(ctx, id)` | `POST /chatbots/{id}/train` | Start chatbot training |
| `client.Chatbots.GetTrainStatus(ctx, id)` | `GET /chatbots/{id}/train-status` | Get chatbot training status |
| `client.Chatbots.SendMessage(ctx, id, body)` | `POST /chatbots/{id}/messages` | Send chatbot message |
| `client.Chatbots.ListConversations(ctx, id, query)` | `GET /chatbots/{id}/conversations` | List chatbot conversations |
| `client.Chatbots.GetConversationMessages(ctx, id, convId)` | `GET /chatbots/{id}/conversations/{conv_id}/messages` | Get conversation messages |

### Comments

| Method | Endpoint | Description |
| --- | --- | --- |
| `client.Comments.List(ctx, query)` | `GET /social/comments` | List comments |
| `client.Comments.ReplyTo(ctx, commentId, body)` | `POST /social/comments/{comment_id}/reply` | Reply to a comment |
| `client.Comments.Hide(ctx, commentId)` | `POST /social/comments/{comment_id}/hide` | Hide or unhide a comment |
| `client.Comments.Delete(ctx, commentId)` | `DELETE /social/comments/{comment_id}` | Delete a comment |

### Content

| Method | Endpoint | Description |
| --- | --- | --- |
| `client.Content.Rewrite(ctx, body)` | `POST /content/rewrite` | Rewrite content |
| `client.Content.GenerateCaption(ctx, body)` | `POST /content/caption` | Generate a social caption |

### CRM Contacts

| Method | Endpoint | Description |
| --- | --- | --- |
| `client.Contacts.List(ctx, query)` | `GET /contacts` | List contacts |
| `client.Contacts.Create(ctx, body)` | `POST /contacts` | Create or upsert a contact |
| `client.Contacts.Get(ctx, id)` | `GET /contacts/{id}` | Get a contact |
| `client.Contacts.Update(ctx, id, body)` | `PATCH /contacts/{id}` | Update a contact |
| `client.Contacts.AddTags(ctx, id, body)` | `POST /contacts/{id}/tags` | Add tags to a contact |
| `client.Contacts.RemoveTags(ctx, id, body)` | `DELETE /contacts/{id}/tags` | Remove tags from a contact |
| `client.Contacts.ListNotes(ctx, id)` | `GET /contacts/{id}/notes` | List contact notes |
| `client.Contacts.AddNote(ctx, id, body)` | `POST /contacts/{id}/notes` | Add a note to a contact |
| `client.Contacts.Enroll(ctx, id, body)` | `POST /contacts/{id}/enroll` | Enroll a contact in an automation |
| `client.Contacts.AddMessage(ctx, id, body)` | `POST /contacts/{id}/messages` | Log a message on a contact's timeline |

### CRM Custom Fields

| Method | Endpoint | Description |
| --- | --- | --- |
| `client.CustomFields.List(ctx)` | `GET /custom-fields` | List custom fields |
| `client.CustomFields.Create(ctx, body)` | `POST /custom-fields` | Create a custom field |
| `client.CustomFields.Delete(ctx, id)` | `DELETE /custom-fields/{id}` | Delete a custom field |

### CRM Opportunities

| Method | Endpoint | Description |
| --- | --- | --- |
| `client.Opportunities.ListPipelines(ctx)` | `GET /pipelines` | List pipelines |
| `client.Opportunities.CreatePipeline(ctx, body)` | `POST /pipelines` | Create a pipeline |
| `client.Opportunities.List(ctx, query)` | `GET /opportunities` | List opportunities |
| `client.Opportunities.Create(ctx, body)` | `POST /opportunities` | Create an opportunity |
| `client.Opportunities.Get(ctx, id)` | `GET /opportunities/{id}` | Get an opportunity |
| `client.Opportunities.Update(ctx, id, body)` | `PATCH /opportunities/{id}` | Update an opportunity |
| `client.Opportunities.Delete(ctx, id)` | `DELETE /opportunities/{id}` | Delete an opportunity |
| `client.Opportunities.UpdateStatus(ctx, id, body)` | `POST /opportunities/{id}/status` | Update opportunity status |

### Direct Messages

| Method | Endpoint | Description |
| --- | --- | --- |
| `client.Messages.ListConversations(ctx, query)` | `GET /social/conversations` | List DM conversations |
| `client.Messages.List(ctx, conversationId, query)` | `GET /social/conversations/{conversation_id}/messages` | List messages in a conversation |
| `client.Messages.Send(ctx, conversationId, body)` | `POST /social/conversations/{conversation_id}/messages` | Send a direct message |
| `client.Messages.MarkConversationRead(ctx, conversationId)` | `POST /social/conversations/{conversation_id}/read` | Mark a conversation read |

### Images

| Method | Endpoint | Description |
| --- | --- | --- |
| `client.Images.Generate(ctx, body)` | `POST /images/generate` | Generate image |
| `client.Images.List(ctx, query)` | `GET /images` | List images |
| `client.Images.Get(ctx, imageId)` | `GET /images/{image_id}` | Get image |
| `client.Images.Delete(ctx, imageId)` | `DELETE /images/{image_id}` | Delete image |

### Jobs

| Method | Endpoint | Description |
| --- | --- | --- |
| `client.Jobs.List(ctx, query)` | `GET /jobs` | List jobs |
| `client.Jobs.Get(ctx, jobId)` | `GET /jobs/{job_id}` | Get job |
| `client.Jobs.Cancel(ctx, jobId, body)` | `POST /jobs/{job_id}/cancel` | Cancel job |

### Media

| Method | Endpoint | Description |
| --- | --- | --- |
| `client.Media.List(ctx, query)` | `GET /media` | List media |
| `client.Media.Get(ctx, mediaId)` | `GET /media/{media_id}` | Get media |
| `client.Media.Delete(ctx, mediaId)` | `DELETE /media/{media_id}` | Delete media |
| `client.Media.GetUploadUrl(ctx, body)` | `POST /media/upload-url` | Get presigned upload URL |

### Presentations

| Method | Endpoint | Description |
| --- | --- | --- |
| `client.Presentations.Generate(ctx, body)` | `POST /presentations/generate` | Generate presentation |
| `client.Presentations.List(ctx, query)` | `GET /presentations` | List presentations |
| `client.Presentations.Get(ctx, presentationId)` | `GET /presentations/{presentation_id}` | Get presentation |
| `client.Presentations.Delete(ctx, presentationId)` | `DELETE /presentations/{presentation_id}` | Delete presentation |

### Profiles

| Method | Endpoint | Description |
| --- | --- | --- |
| `client.Profiles.List(ctx, query)` | `GET /profiles` | List profiles |
| `client.Profiles.Create(ctx, body)` | `POST /profiles` | Create a profile |
| `client.Profiles.Get(ctx, id)` | `GET /profiles/{id}` | Get a profile |
| `client.Profiles.Delete(ctx, id, body)` | `DELETE /profiles/{id}` | Delete a profile |
| `client.Profiles.ListAccounts(ctx, id)` | `GET /profiles/{id}/accounts` | List a profile's connected accounts |
| `client.Profiles.Pause(ctx, id)` | `POST /profiles/{id}/pause` | Pause a profile |
| `client.Profiles.Resume(ctx, id)` | `POST /profiles/{id}/resume` | Resume a profile |
| `client.Profiles.CreateConnectLink(ctx, id, body)` | `POST /profiles/{id}/connect-link` | Create a hosted connect link |
| `client.Profiles.CreateConnectUrl(ctx, id, platform, body)` | `POST /profiles/{id}/connect/{platform}` | Get a raw connect URL for one platform |
| `client.Profiles.GetAccountBilling(ctx)` | `GET /me/account-billing` | Account billing summary |

### SEO

| Method | Endpoint | Description |
| --- | --- | --- |
| `client.SEO.KeywordResearch(ctx, body)` | `POST /seo/keyword-research` | Keyword research |
| `client.SEO.Serp(ctx, body)` | `POST /seo/serp` | Live SERP lookup |
| `client.SEO.KeywordDifficulty(ctx, body)` | `POST /seo/keyword-difficulty` | Keyword difficulty |
| `client.SEO.RankedKeywords(ctx, body)` | `POST /seo/ranked-keywords` | Ranked keywords (rank tracking) |
| `client.SEO.DomainOverview(ctx, body)` | `POST /seo/domain-overview` | Domain rank overview |
| `client.SEO.Competitors(ctx, body)` | `POST /seo/competitors` | Organic competitors |
| `client.SEO.BacklinksSummary(ctx, body)` | `POST /seo/backlinks-summary` | Backlink profile summary |
| `client.SEO.Audit(ctx, body)` | `POST /seo/audit` | On-page SEO audit |
| `client.SEO.BacklinkProspects(ctx, body)` | `POST /seo/backlink-prospects` | Backlink prospects (link gap) |
| `client.SEO.ReferringDomains(ctx, body)` | `POST /seo/referring-domains` | Referring domains |
| `client.SEO.BacklinkAnchors(ctx, body)` | `POST /seo/backlink-anchors` | Backlink anchors |
| `client.SEO.SpamScore(ctx, body)` | `POST /seo/spam-score` | Backlink spam score |
| `client.SEO.RankHistory(ctx, body)` | `POST /seo/rank-history` | Historical rank overview |
| `client.SEO.SiteAudit(ctx, body)` | `POST /seo/site-audit` | Deep site audit |
| `client.SEO.BrandLookup(ctx, body)` | `POST /seo/brand-lookup` | AI Visibility: brand lookup |
| `client.SEO.PromptExplorer(ctx, body)` | `POST /seo/prompt-explorer` | AI Visibility: prompt explorer |
| `client.SEO.AiAudit(ctx, body)` | `POST /seo/ai-audit` | AI Visibility Audit (async) |

### Shorts

| Method | Endpoint | Description |
| --- | --- | --- |
| `client.Shorts.Generate(ctx, body)` | `POST /shorts/generate` | Generate viral shorts from a long video |
| `client.Shorts.List(ctx, query)` | `GET /shorts` | List shorts jobs |
| `client.Shorts.Get(ctx, uid)` | `GET /shorts/{uid}` | Get shorts job + clips |

### Social

| Method | Endpoint | Description |
| --- | --- | --- |
| `client.Social.ListAccounts(ctx)` | `GET /social/accounts` | List social accounts |
| `client.Social.ListPosts(ctx, query)` | `GET /social/posts` | List social posts |
| `client.Social.CreatePost(ctx, body)` | `POST /social/posts` | Create post (publish immediately) |
| `client.Social.SchedulePost(ctx, body)` | `POST /social/posts/schedule` | Schedule post |
| `client.Social.GetPost(ctx, postId)` | `GET /social/posts/{post_id}` | Get social post |
| `client.Social.UpdatePost(ctx, postId, body)` | `PATCH /social/posts/{post_id}` | Update social post |
| `client.Social.DeletePost(ctx, postId)` | `DELETE /social/posts/{post_id}` | Delete social post |
| `client.Social.UpdateAccount(ctx, accountId, body)` | `PATCH /social/accounts/{account_id}` | Rename account |
| `client.Social.GetAccountHealth(ctx, accountId)` | `GET /social/accounts/{account_id}/health` | Account health |
| `client.Social.GetAccountReconnectUrl(ctx, accountId)` | `GET /social/accounts/{account_id}/reconnect-url` | Account reconnect URL |
| `client.Social.PauseAccount(ctx, accountId)` | `POST /social/accounts/{account_id}/pause` | Pause posting to an account |
| `client.Social.ResumeAccount(ctx, accountId)` | `POST /social/accounts/{account_id}/resume` | Resume posting to an account |
| `client.Social.RetryPost(ctx, postId, body)` | `POST /social/posts/{post_id}/retry` | Retry publishing a post |
| `client.Social.ConnectAccountStatus(ctx, platform)` | `GET /social/connect/{platform}` | Poll headless connection status |
| `client.Social.ConnectAccount(ctx, platform, body)` | `POST /social/connect/{platform}` | Start headless account connection |
| `client.Social.ListQueues(ctx)` | `GET /social/queues` | List queues |
| `client.Social.CreateQueue(ctx, body)` | `POST /social/queues` | Create queue |
| `client.Social.GetQueue(ctx, queueId)` | `GET /social/queues/{queue_id}` | Get queue |
| `client.Social.UpdateQueue(ctx, queueId, body)` | `PUT /social/queues/{queue_id}` | Update queue |
| `client.Social.DeleteQueue(ctx, queueId)` | `DELETE /social/queues/{queue_id}` | Delete queue |
| `client.Social.GetQueueNextSlot(ctx, queueId)` | `GET /social/queues/{queue_id}/next-slot` | Get next open slot |
| `client.Social.PreviewQueueSlots(ctx, queueId, query)` | `GET /social/queues/{queue_id}/preview` | Preview upcoming slots |
| `client.Social.UnpublishPost(ctx, postId, body)` | `POST /social/posts/{post_id}/unpublish` | Unpublish post |
| `client.Social.ValidatePost(ctx, body)` | `POST /social/validate/post` | Validate post content |
| `client.Social.ValidateMedia(ctx, body)` | `POST /social/validate/media` | Validate media URL |
| `client.Social.StopPostRecycle(ctx, postId)` | `DELETE /social/posts/{post_id}/recycle` | Stop recycling |
| `client.Social.BulkSchedulePosts(ctx, body)` | `POST /social/posts/bulk` | Bulk schedule posts |
| `client.Social.ValidateBulkBatch(ctx, body)` | `POST /social/posts/bulk/validate` | Validate a bulk batch |
| `client.Social.BulkAccountHealth(ctx)` | `GET /social/accounts/health` | Bulk account health |
| `client.Social.AccountFollowerStats(ctx, query)` | `GET /social/accounts/follower-stats` | Follower stats |
| `client.Social.TiktokCreatorInfo(ctx, accountId)` | `GET /social/accounts/{account_id}/tiktok/creator-info` | TikTok creator info |
| `client.Social.MoveAccount(ctx, accountId, body)` | `POST /social/accounts/{account_id}/move` | Move account to profile |
| `client.Social.ListAccountGroups(ctx)` | `GET /social/account-groups` | List account groups |
| `client.Social.CreateAccountGroup(ctx, body)` | `POST /social/account-groups` | Create account group |
| `client.Social.GetAccountGroup(ctx, groupId)` | `GET /social/account-groups/{group_id}` | Get account group |
| `client.Social.UpdateAccountGroup(ctx, groupId, body)` | `PUT /social/account-groups/{group_id}` | Update account group |
| `client.Social.DeleteAccountGroup(ctx, groupId)` | `DELETE /social/account-groups/{group_id}` | Delete account group |

### URLs

| Method | Endpoint | Description |
| --- | --- | --- |
| `client.URLs.Shorten(ctx, body)` | `POST /urls/shorten` | Shorten URL |
| `client.URLs.List(ctx, query)` | `GET /urls` | List short URLs |
| `client.URLs.Get(ctx, urlId)` | `GET /urls/{url_id}` | Get short URL |
| `client.URLs.Delete(ctx, urlId)` | `DELETE /urls/{url_id}` | Delete short URL |
| `client.URLs.GetStats(ctx, urlId)` | `GET /urls/{url_id}/stats` | Get short URL stats |

### Videos

| Method | Endpoint | Description |
| --- | --- | --- |
| `client.Videos.ListModels(ctx)` | `GET /videos/models` | List available video models |
| `client.Videos.Generate(ctx, body)` | `POST /videos/generate` | Generate video |
| `client.Videos.List(ctx, query)` | `GET /videos` | List videos |
| `client.Videos.Get(ctx, videoId)` | `GET /videos/{video_id}` | Get video |
| `client.Videos.Delete(ctx, videoId)` | `DELETE /videos/{video_id}` | Delete video |
| `client.Videos.GenerateHook(ctx, body)` | `POST /videos/hook` | Generate a viral hook line |
| `client.Videos.SuggestBroll(ctx, body)` | `POST /videos/broll-suggest` | Suggest B-roll moments |
| `client.Videos.SuggestEmphasis(ctx, body)` | `POST /videos/emphasis` | Suggest on-screen emphasis |
| `client.Videos.GenerateViralThumbnail(ctx, body)` | `POST /videos/viral-thumbnail` | Generate a viral thumbnail |

### Webhooks

| Method | Endpoint | Description |
| --- | --- | --- |
| `client.Webhooks.List(ctx)` | `GET /webhooks` | List webhooks |
| `client.Webhooks.Create(ctx, body)` | `POST /webhooks` | Create webhook |
| `client.Webhooks.Update(ctx, id, body)` | `PUT /webhooks/{id}` | Update webhook |
| `client.Webhooks.Delete(ctx, id)` | `DELETE /webhooks/{id}` | Delete webhook |
| `client.Webhooks.ListLogs(ctx, query)` | `GET /webhooks/logs` | List webhook delivery logs |
| `client.Webhooks.Test(ctx, id)` | `POST /webhooks/{id}/test` | Send test webhook |

### Workspaces

| Method | Endpoint | Description |
| --- | --- | --- |
| `client.Workspaces.List(ctx)` | `GET /workspaces` | List workspaces (sub-accounts) |
| `client.Workspaces.Create(ctx, body)` | `POST /workspaces` | Create a workspace (sub-account) |
| `client.Workspaces.BulkAction(ctx, body)` | `POST /workspaces/bulk` | Bulk sub-account action |
| `client.Workspaces.Get(ctx, id)` | `GET /workspaces/{id}` | Get a workspace (sub-account) |
| `client.Workspaces.Delete(ctx, id, body)` | `DELETE /workspaces/{id}` | Delete a workspace (sub-account) |
| `client.Workspaces.DisableSaas(ctx, id, body)` | `POST /workspaces/{id}/disable-saas` | Disable SaaS mode for a workspace |
| `client.Workspaces.Pause(ctx, id)` | `POST /workspaces/{id}/pause` | Pause (suspend) a workspace |
| `client.Workspaces.Resume(ctx, id)` | `POST /workspaces/{id}/resume` | Resume a paused workspace |
| `client.Workspaces.GetSubscription(ctx, id)` | `GET /workspaces/{id}/subscription` | Get a sub-account's subscription |
| `client.Workspaces.GetWallet(ctx, id)` | `GET /workspaces/{id}/wallet` | Get a sub-account's wallet balance |
| `client.Workspaces.ListSaasPlans(ctx)` | `GET /saas/plans` | List SaaS plans |
| `client.Workspaces.GetSaasPlan(ctx, id)` | `GET /saas/plans/{id}` | Get a SaaS plan |
<!-- END GENERATED REFERENCE -->

## Regeneration

This SDK is generated from the [SmartlyQ OpenAPI spec](https://docs.smartlyq.com). When the spec changes, CI regenerates the client, README, and tests, bumps the version, tags a release, and Go module proxies pick it up automatically.

## License

MIT
