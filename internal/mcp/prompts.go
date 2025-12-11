// ABOUTME: MCP prompt definitions and handlers
// ABOUTME: Provides workflow templates for RSS digest operations

package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerPrompts() {
	s.registerDailyDigestPrompt()
	s.registerCatchUpPrompt()
	s.registerCurateFeedsPrompt()
}

func (s *Server) registerDailyDigestPrompt() {
	s.mcpServer.AddPrompt(
		mcp.Prompt{
			Name:        "daily-digest",
			Description: "Generate a summary of today's feed entries to catch up on the latest content from your subscriptions",
			Arguments:   []mcp.PromptArgument{},
		},
		s.handleDailyDigest,
	)
}

//nolint:funlen // Prompt handlers contain large template strings
func (s *Server) handleDailyDigest(_ context.Context, _ mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	template := `# Daily Digest

## Overview
Review and summarize today's feed entries to stay up-to-date with your RSS/Atom subscriptions. This workflow helps you quickly digest the most important content published today across all your feeds.

## When to Use
- Daily morning routine to catch up on overnight content
- End of day review to see what you missed
- After syncing feeds to see new content
- When you want a quick overview without reading every entry

## Workflow Steps

### Step 1: Check Today's Statistics
Get an overview of today's activity across all feeds.

**Use digest://stats resource:**
- Review total entries published today
- Check per-feed breakdown
- Identify which feeds are most active

**Example:**
- digest://stats shows 45 new entries today
- Top feeds: Tech Blog (12), News Feed (15), Dev Community (8)

### Step 2: Scan Today's Entries
Review the full list of entries published today.

**Use digest://entries/today resource:**
- See all entries from the past 24 hours
- Sort by feed or publication time
- Note titles, authors, and brief excerpts

**What to look for:**
- Breaking news or time-sensitive content
- Topics matching your current interests
- High-value content from trusted sources

**Example:**
- digest://entries/today → 45 entries
- Scan titles for keywords: "release", "announcement", "breaking"
- Identify 5-10 must-read items

### Step 3: Prioritize Content
Group entries by importance and relevance.

**Categorize by priority:**
- **Must read now:** Breaking news, urgent updates, deadline-sensitive content
- **Read today:** High-value content, trending topics, important analysis
- **Read later:** Interesting but not time-sensitive, tutorials, long-form
- **Skip:** Low-priority, duplicate coverage, off-topic

**Example prioritization:**
- Must read (3 entries): Product launch announcement, security advisory, industry news
- Read today (7 entries): Analysis pieces, feature announcements, community discussions
- Read later (20 entries): Tutorials, opinion pieces, in-depth guides
- Skip (15 entries): Duplicate coverage, off-topic, low-signal

### Step 4: Read High-Priority Content
Focus on must-read and high-priority items.

**Reading strategy:**
- Start with must-read items
- Scan headlines and summaries for read-today items
- Use mark_entry_read tool as you go
- Take notes on important insights

**Tips:**
- Set a time limit (e.g., 30 minutes)
- Focus on unique insights, skip duplicate coverage
- Mark entries read even if you skim (keeps tracking accurate)

### Step 5: Generate Summary
Create a brief digest of key takeaways.

**Summary structure:**
- **Top Stories:** 2-3 most important items with key points
- **Notable Updates:** 3-5 significant but not urgent items
- **Trending Topics:** Themes or patterns across multiple entries
- **Action Items:** Follow-ups, things to investigate, or share

**Example summary:**

    Daily Digest - December 11, 2025

    Top Stories:
    1. Major Framework v5.0 Released - Breaking changes to API, migration guide available
    2. Security Advisory: CVE-2025-1234 - Patch available, affects production systems
    3. Industry Analysis: AI Coding Tools Adoption Study - 67%% of devs now use daily

    Notable Updates:
    - New database features announced at conference
    - Tutorial on performance optimization published
    - Community discussion on best practices heating up

    Trending: AI/ML tools, performance optimization, security updates

    Action Items:
    - Review Framework v5.0 migration guide
    - Apply security patch to production
    - Share AI adoption study with team

### Step 6: Mark Entries and Clean Up
Update read status for processed entries.

**Use mark_entry_read tool:**
- Mark all read entries (even if skimmed)
- Leave unread items you want to revisit
- Accurate tracking helps with future catch-up

**Clean-up actions:**
- Mark must-read and read-today items as read
- Leave read-later items unread for future sessions
- Check digest://entries/unread to see remaining backlog

## Tips and Best Practices
- **Consistent timing:** Run daily digest at the same time each day (morning or evening)
- **Time-box it:** Set a limit (15-30 minutes) to avoid rabbit holes
- **Trust your feeds:** If a feed consistently provides low-value content, unsubscribe
- **Themes over individual items:** Look for patterns across multiple entries
- **Mark as read liberally:** Better to mark read and maintain clean state
- **Focus on unique insights:** Skip duplicate coverage of same story
- **Use resources efficiently:** digest://entries/today is faster than filtering manually

## Integration with Other Workflows
- **After sync:** Run daily-digest after using sync_feeds tool
- **Before catch-up:** Use daily-digest for recent content, catch-up for backlog
- **With curate-feeds:** Use digest to identify low-value feeds to remove

## Example Daily Digest Session

**Step 1: Stats**
- digest://stats → 38 new entries today across 8 feeds
- Most active: HackerNews (15), Tech Blog (8), Dev.to (6)

**Step 2: Scan**
- digest://entries/today → Review all 38 entries
- Scan titles and authors
- Note 3-4 particularly interesting items

**Step 3: Prioritize**
- Must read (2): Security advisory, product launch
- Read today (5): Analysis pieces, feature announcements
- Read later (18): Tutorials, long-form
- Skip (13): Duplicate HackerNews discussions

**Step 4: Read**
- 15 minutes: Read 2 must-read items, skim 5 read-today items
- Mark 7 entries as read using mark_entry_read
- Take notes on security advisory (action required)

**Step 5: Summary**
Generated concise summary with top 3 stories, trending topics, 1 action item

**Step 6: Cleanup**
- Marked 20 entries as read (7 fully read, 13 skipped)
- Left 18 entries unread for later
- Backlog status: digest://entries/unread shows 156 total unread (manageable)

**Ready to create your daily digest?**
1. Check digest://stats for today's overview
2. Review digest://entries/today for all new content
3. Prioritize by importance and relevance
4. Read high-priority items
5. Generate summary with key takeaways
6. Mark entries read and clean up
`

	return &mcp.GetPromptResult{
		Description: "Daily digest workflow for today's feed entries",
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.TextContent{
					Type: "text",
					Text: template,
				},
			},
		},
	}, nil
}

func (s *Server) registerCatchUpPrompt() {
	s.mcpServer.AddPrompt(
		mcp.Prompt{
			Name:        "catch-up",
			Description: "Catch up on missed entries from recent days when you've fallen behind on your RSS feeds",
			Arguments: []mcp.PromptArgument{
				{
					Name:        "days",
					Description: "Number of days to catch up on (default: 7)",
					Required:    false,
				},
			},
		},
		s.handleCatchUp,
	)
}

//nolint:funlen // Prompt handlers contain large template strings
func (s *Server) handleCatchUp(_ context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	days := "7"
	if req.Params.Arguments != nil {
		if d, ok := req.Params.Arguments["days"]; ok && d != "" {
			days = d
		}
	}

	template := fmt.Sprintf(`# Catch Up on Missed Entries

## Overview
Efficiently process a backlog of unread entries from the past %s days when you've fallen behind on your RSS subscriptions. This workflow helps you triage, prioritize, and quickly get back to inbox zero without reading every single item.

## When to Use
- After vacation or time away from feeds
- When unread count has grown unmanageable
- Weekly cleanup to process accumulated entries
- After subscribing to several new feeds at once
- When digest://stats shows large unread backlog

## Workflow Steps

### Step 1: Assess the Backlog
Understand the scope of what you're catching up on.

**Use digest://stats resource:**
- Check total unread count
- Review per-feed breakdown
- Identify which feeds have the most unread

**Questions to answer:**
- How many total unread entries?
- Which feeds are the biggest contributors?
- Are any feeds consistently high-volume low-value?

**Example:**
- digest://stats → 347 unread entries (yikes!)
- Top contributors: HackerNews (156), Reddit Dev (89), Tech Blog (45)
- Observation: HackerNews and Reddit are very high-volume

### Step 2: Set Realistic Goals
Be honest about what you can process in %s days of catch-up.

**Triage strategy:**
- **Can't read everything:** Accept that you'll skip most items
- **Focus on high-value:** Prioritize feeds with best signal-to-noise
- **Time-box it:** Allocate 30-60 minutes total, not per feed
- **Aim for progress:** Reducing backlog by 50%% is success

**Goal setting:**
- Total backlog: 347 entries
- Time available: 45 minutes
- Realistic goal: Process top 3 feeds (~200 entries), mark rest as read
- Acceptable outcome: Identify 10-15 must-read items, skip the rest

### Step 3: Process High-Value Feeds First
Start with feeds that consistently provide the best content.

**For each high-value feed:**
1. Use list_entries with feed_id filter
2. Scan titles and dates quickly
3. Mark interesting items for reading
4. Bulk mark the rest as read

**Processing strategy per feed:**
- **30 seconds:** Scan all titles
- **2 minutes:** Read summaries of interesting items
- **5 minutes:** Deep read 1-2 must-read items
- **Bulk action:** Mark remaining entries as read

**Example - Tech Blog (45 entries):**
- Scan: 30 seconds to review all titles
- Identify: 5 interesting items
- Read: 5 minutes on 2 critical posts
- Mark read: Remaining 40 entries (use mark_entry_read)

### Step 4: Handle High-Volume Low-Signal Feeds
Deal with feeds that produce lots of content but limited value.

**Strategies:**
- **Skim mode:** Read only titles, mark all as read
- **Spot check:** Read 2-3 recent items, mark rest as read
- **Declare bankruptcy:** Mark entire feed's backlog as read
- **Consider unsubscribing:** Use curate-feeds workflow

**High-volume feed processing (e.g., HackerNews - 156 entries):**
- Don't read everything - impossible and unnecessary
- Scan last 2 days of titles (maybe 30 entries)
- Pick 2-3 most interesting discussions
- Mark all 156 as read and move on

**Feed bankruptcy:**
When a feed has >100 unread and low value:
1. Accept you won't read them
2. Mark all as read in bulk
3. Start fresh from today
4. Consider if you really need this subscription

### Step 5: Identify Must-Read Content
Extract the truly important items from the backlog.

**Criteria for must-read:**
- **Time-sensitive:** Security advisories, breaking news, deadlines
- **High relevance:** Directly applicable to current work/interests
- **Unique insights:** Content you can't get elsewhere
- **Trusted sources:** Known high-quality authors

**NOT must-read:**
- Duplicate coverage of same story
- Generic tutorials (can find anytime)
- Opinion pieces (unless exceptional)
- Old news (already happened, you missed it, move on)

**Example must-read list:**
From 347 unread, identified:
1. Security patch announcement (Tech Blog)
2. Framework release notes (Dev Community)
3. Industry analysis piece (Newsletter)
4. Interview with expert (Podcast Blog)
Total: 4 items to actually read deeply

### Step 6: Execute Bulk Actions
Efficiently mark processed content as read.

**Use mark_entry_read tool:**
- Mark must-read items as read after reading
- Bulk mark skipped feeds as read
- Leave a few interesting items unread if you want to revisit

**Bulk processing tips:**
- Process by feed (mark entire feed at once)
- Use list_entries to get entry IDs
- Mark in batches to avoid rate limits

**Example bulk actions:**
- HackerNews: Mark all 156 as read (bankruptcy)
- Reddit Dev: Mark all 89 as read (low signal)
- Tech Blog: Mark 43 as read (kept 2 unread)
- Newsletters: Mark 30 as read (kept 4 must-read)
- Result: 347 → 6 unread (clean slate!)

### Step 7: Create Catch-Up Summary
Document what you learned and what needs follow-up.

**Summary structure:**
- **Backlog processed:** Starting count → ending count
- **Must-read items:** List of truly important content
- **Key takeaways:** Main themes or insights
- **Action items:** Follow-ups from catch-up session
- **Feed health:** Notes on which feeds to keep/remove

**Example summary:**

    Catch-Up Summary - %s Days

    Processed: 347 → 6 unread (98%%%% reduction)

    Must-Read Items:
    1. [Tech Blog] Security Advisory CVE-2025-5678 - Action required
    2. [Dev Community] Framework 6.0 Release - Migration needed
    3. [Newsletter] State of Developer Tools 2025 - Informative
    4. [Podcast] Expert Interview on AI - Worth listening

    Key Takeaways:
    - Security patches needed urgently
    - Major framework upgrade coming, plan migration
    - AI tools becoming industry standard

    Action Items:
    - [ ] Apply security patch by Friday
    - [ ] Schedule framework upgrade planning meeting
    - [ ] Review AI tools for team evaluation

    Feed Health:
    - Keep: Tech Blog, Dev Community, Newsletter (high signal)
    - Consider removing: HackerNews, Reddit Dev (high volume, low personal relevance)
    - Total feeds: 8, manageable if processed regularly

## Tips and Best Practices
- **Don't aim for perfection:** Catching up means strategic skipping
- **Trust your instincts:** If a title doesn't grab you, skip it
- **Declare bankruptcy when needed:** Better to mark all read and start fresh
- **Time-box ruthlessly:** Set timer, stop when it goes off
- **Focus on recency:** Recent content is more valuable than old
- **Use feed health insights:** Catch-up reveals which feeds are worth keeping
- **Prevent future backlog:** Unsubscribe from consistently low-value feeds
- **Regular cadence:** Weekly catch-up prevents massive backlogs

## Anti-Patterns to Avoid
- ❌ Trying to read every entry (impossible and exhausting)
- ❌ Starting with low-value high-volume feeds (waste of time)
- ❌ Reading old news that's no longer relevant
- ❌ Keeping feeds "just in case" (unsubscribe!)
- ❌ Feeling guilty about marking as read (it's a tool, not a moral obligation)
- ❌ Processing feeds alphabetically (prioritize by value)
- ❌ Saving items "to read later" that you'll never read (be honest)

## Example Catch-Up Session (%s Days)

**Step 1: Assess**
- digest://stats → 347 unread across 8 feeds
- Breakdown: HackerNews (156), Reddit (89), Tech Blog (45), Others (57)

**Step 2: Goals**
- Time budget: 45 minutes
- Goal: Reduce to <10 unread
- Strategy: Process top 3 feeds, bankruptcy on high-volume

**Step 3: High-Value Feeds (20 minutes)**
- Tech Blog: 5 min → Found 2 must-reads
- Dev Community: 5 min → Found 1 must-read
- Newsletter: 5 min → Found 1 must-read
- Others: 5 min → Scanned, found nothing critical

**Step 4: High-Volume Feeds (5 minutes)**
- HackerNews: Declared bankruptcy, marked all 156 as read
- Reddit Dev: Declared bankruptcy, marked all 89 as read

**Step 5: Must-Read**
Identified 4 critical items from 347 total

**Step 6: Bulk Actions (5 minutes)**
- Marked 341 entries as read
- Kept 6 items unread (4 must-read, 2 interesting)

**Step 7: Summary (5 minutes)**
Documented findings, action items, feed health insights

**Result: 347 → 6 unread in 40 minutes!**

**Ready to catch up?**
1. Check digest://stats to assess backlog size
2. Set realistic goals (time and coverage)
3. Process high-value feeds first
4. Declare bankruptcy on high-volume low-signal feeds
5. Identify must-read content
6. Bulk mark processed entries as read
7. Create summary with action items and feed health notes
`, days, days, days, days)

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Catch-up workflow for processing %s days of unread entries", days),
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.TextContent{
					Type: "text",
					Text: template,
				},
			},
		},
	}, nil
}

func (s *Server) registerCurateFeedsPrompt() {
	s.mcpServer.AddPrompt(
		mcp.Prompt{
			Name:        "curate-feeds",
			Description: "Review your feed subscriptions and optimize them by removing low-value feeds and finding better sources",
			Arguments:   []mcp.PromptArgument{},
		},
		s.handleCurateFeeds,
	)
}

//nolint:funlen // Prompt handlers contain large template strings
func (s *Server) handleCurateFeeds(_ context.Context, _ mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	template := `# Curate Feeds

## Overview
Systematically review and optimize your RSS/Atom feed subscriptions to maintain a high signal-to-noise ratio. Remove feeds that no longer provide value, identify gaps in coverage, and discover better sources. A well-curated feed list makes daily digest and catch-up workflows much more effective.

## When to Use
- Feed backlog consistently grows despite regular catch-up
- Many unread entries but few interesting items
- Quarterly or monthly feed hygiene routine
- After trying several new feeds, need to decide which to keep
- Feeling overwhelmed by total feed volume
- When digest://stats shows concerning patterns

## Workflow Steps

### Step 1: Analyze Current Feed Health
Get quantitative data on all subscriptions.

**Use digest://stats resource:**
- Review total feed count
- Check per-feed entry counts
- Identify unread distribution
- Note error counts and fetch failures

**Key metrics to track:**
- **Volume:** Entries per feed per week
- **Read rate:** % of entries you actually read
- **Value ratio:** Must-read items / total entries
- **Freshness:** Last successful fetch time
- **Errors:** Persistent fetch failures

**Example analysis:**

    Total feeds: 12
    Total unread: 423

    Per-feed breakdown:
    1. HackerNews: 187 unread, 0%% read (HIGH VOLUME, LOW VALUE)
    2. Tech Blog: 45 unread, 60%% read (GOOD VALUE)
    3. Dev.to: 89 unread, 10%% read (HIGH VOLUME, LOW VALUE)
    4. Newsletter: 12 unread, 90%% read (EXCELLENT VALUE)
    5. Podcast Blog: 8 unread, 75%% read (GOOD VALUE)
    6. Generic News: 34 unread, 5%% read (LOW VALUE)
    7. Dead Feed: 0 unread, 0 errors, last fetch 45 days ago (INACTIVE)
    8. [Continue for all feeds...]

    Patterns:
    - 3 feeds producing 70%% of volume
    - 2 feeds with <10%% read rate
    - 1 feed hasn't updated in 6 weeks

### Step 2: Categorize Feeds
Group feeds by value and usage patterns.

**Categories:**

**✅ Keep (High Value):**
- High read rate (>50%)
- Consistently interesting content
- Unique perspective or information
- Manageable volume

**⚠️ Probation (Uncertain):**
- Recently added, need more data
- Inconsistent quality
- Moderate read rate (20-50%)
- Might improve with selective reading

**❌ Remove (Low Value):**
- Very low read rate (<10%)
- High volume, low signal
- Duplicate coverage with better feeds
- No longer relevant to interests
- Persistent fetch errors
- Inactive (no new entries in 30+ days)

**Example categorization:**
- Keep (5): Tech Blog, Newsletter, Podcast Blog, Expert Blog, Industry News
- Probation (3): Dev.to, New Feed, Experimental Source
- Remove (4): HackerNews, Generic News, Dead Feed, Duplicate Coverage

### Step 3: Evaluate Each Feed
Review feeds systematically, starting with candidates for removal.

**For each feed:**
1. Review recent entries (use list_entries with feed_id)
2. Check how many you read vs skipped
3. Identify if content is unique or duplicated
4. Consider if it aligns with current interests
5. Decide: Keep, Probation, or Remove

**Evaluation questions:**
- **Value:** Has this feed taught me something valuable in past month?
- **Uniqueness:** Do I get this information elsewhere?
- **Relevance:** Still aligned with my interests/work?
- **Volume:** Is the entry volume manageable?
- **Quality:** Is signal-to-noise ratio acceptable?
- **Timeliness:** Are updates still regular and fresh?

**Example evaluation - HackerNews:**
- Volume: ~25 entries/day = 175/week (VERY HIGH)
- Read rate: 0% (skipped all 187 unread)
- Uniqueness: Duplicate coverage with other tech feeds
- Value: Interesting discussions but overwhelming volume
- Decision: REMOVE - Getting news from better-curated sources

### Step 4: Remove Low-Value Feeds
Unsubscribe from feeds that don't make the cut.

**Before removing:**
- Mark all entries as read (cleanup)
- Export feed URL for reference (in case you want to re-add later)
- Document reason for removal (learn from the pattern)

**Use remove_feed tool:**
- remove_feed(feed_id="...")
- Removes feed and all associated entries
- Frees up mental bandwidth

**Example removals:**
- HackerNews: Too high volume, duplicate coverage
- Generic News: Low relevance, better sources exist
- Dead Feed: No updates in 6 weeks, likely abandoned
- Duplicate Coverage: Already getting same info from better feed

### Step 5: Optimize Probation Feeds
Decide strategy for uncertain feeds.

**Strategies:**

**Strategy 1: Time-limited trial**
- Keep for 2-4 weeks
- Track read rate actively
- Review again and decide

**Strategy 2: Selective reading**
- Only read specific topics/authors
- Mark rest as read immediately
- If too much effort, remove

**Strategy 3: Reduce frequency**
- If high volume, check less often
- Process weekly instead of daily
- If still overwhelming, remove

**Example probation decisions:**
- Dev.to: Keep for 2 more weeks, track if quality improves
- New Feed: Interesting but new, give it 1 month trial
- Experimental Source: Too much manual filtering needed, REMOVE

### Step 6: Identify Coverage Gaps
Look for missing topics or perspectives.

**Gap analysis questions:**
- What topics interest me but aren't covered?
- Are there important sources I'm missing?
- Do I have diverse perspectives or echo chamber?
- Any tools/technologies I use without following updates?

**Finding new feeds:**
- Search for topic-specific blogs
- Follow thought leaders' RSS feeds
- Look for official project blogs/announcements
- Ask community for recommendations
- Check OPML directories

**Example gap identification:**
- Missing: Security news (need dedicated feed)
- Missing: Database updates (following but no feed)
- Echo chamber: All feeds have similar perspective
- Action: Add security feed, database blog, contrarian viewpoint

### Step 7: Document Feed Hygiene
Create a record of your curation decisions.

**Documentation to keep:**
- Date of curation
- Feeds removed and why
- Feeds added and why
- Current feed count and target
- Next review date

**Example documentation:**

    Feed Curation - December 11, 2025

    Starting feeds: 12
    Ending feeds: 8 (33%% reduction)

    Removed (4):
    1. HackerNews - High volume (175/week), 0%% read rate, duplicate coverage
    2. Generic News - Low relevance, better sources exist
    3. Dead Feed - No updates since Oct 15, abandoned
    4. Duplicate Tech Feed - Same content as Tech Blog but worse UX

    Added (1):
    1. Security Weekly - Gap in security news coverage

    Kept (7):
    - Tech Blog, Newsletter, Podcast Blog, Expert Blog, Industry News
    - Dev.to (probation - review in 2 weeks)
    - New Feed (probation - review in 4 weeks)

    Target: <10 feeds with >50%% read rate
    Next review: March 1, 2026 (quarterly)

## Tips and Best Practices
- **Quarterly reviews:** Curate every 3 months minimum
- **Ruthless curation:** Better 5 great feeds than 20 mediocre ones
- **Track metrics:** Use read rate as objective measure
- **Quality over quantity:** More feeds ≠ more value
- **Accept FOMO:** Can't follow everything, focus on best sources
- **Evolve with interests:** Remove feeds when interests change
- **Give new feeds time:** 2-4 weeks trial before deciding
- **Export before deleting:** Keep URLs for potential re-subscription
- **Document decisions:** Learn from patterns in removed feeds

## Key Metrics for Feed Health

**Healthy feed list:**
- Total feeds: <15 (manageable daily volume)
- Read rate per feed: >40% (mostly valuable content)
- Unread backlog: <100 (caught up within week)
- Inactive feeds: 0 (all actively publishing)
- Error feeds: 0 (all fetching successfully)

**Warning signs:**
- Total feeds: >25 (too much to process)
- Read rate: <20% on multiple feeds (low value)
- Unread backlog: >500 (overwhelmed)
- Inactive feeds: >2 (deadweight)
- Persistent errors: >1 (feed issues)

## Anti-Patterns to Avoid
- ❌ Keeping feeds "just in case" (be honest about value)
- ❌ Subscribing to everything (quality over quantity)
- ❌ Never removing feeds (interests change, sources change)
- ❌ Feeling obligated to read (feeds serve you, not vice versa)
- ❌ Optimizing for completeness (optimize for value)
- ❌ Avoiding curation due to effort (pays dividends quickly)
- ❌ Not trying new feeds (stagnation is also a problem)

## Example Curation Session

**Step 1: Analyze (10 minutes)**
- digest://stats shows 12 feeds, 423 unread
- Calculate read rates per feed
- Identify 3 high-volume low-value feeds

**Step 2: Categorize (5 minutes)**
- Keep: 5 feeds (high value, good read rate)
- Probation: 3 feeds (uncertain, need more data)
- Remove: 4 feeds (low value, clear decision)

**Step 3: Evaluate (15 minutes)**
- Review recent entries from probation and remove categories
- Check uniqueness of content
- Make final keep/remove decisions

**Step 4: Remove (10 minutes)**
- Mark all entries in removed feeds as read
- Export feed URLs for records
- Remove 4 feeds using remove_feed tool
- Document removal reasons

**Step 5: Optimize Probation (5 minutes)**
- Set 2-week trial for Dev.to
- Decide to remove one probation feed immediately (too much work)
- Keep one probation feed with selective reading strategy

**Step 6: Gaps (10 minutes)**
- Identify security news gap
- Search for and subscribe to Security Weekly feed
- Add to high-value category

**Step 7: Document (5 minutes)**
- Record decisions and rationale
- Set next review date (3 months)
- Update feed management notes

**Result: 12 → 8 feeds, much higher average value!**

**Ready to curate your feeds?**
1. Analyze current feed health with digest://stats
2. Categorize feeds: Keep, Probation, Remove
3. Evaluate each feed systematically
4. Remove low-value feeds
5. Set strategy for probation feeds
6. Identify and fill coverage gaps
7. Document your decisions and next review date
`

	return &mcp.GetPromptResult{
		Description: "Feed curation workflow for optimizing RSS subscriptions",
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.TextContent{
					Type: "text",
					Text: template,
				},
			},
		},
	}, nil
}
