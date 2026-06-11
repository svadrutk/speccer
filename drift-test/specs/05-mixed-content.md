# Mixed Content Test

## Overview
This is just prose about the feature. It mentions `session_id` and `playlist_id` inline
but these should NOT be parsed as field definitions since they're not in a schema section.

## User Stories
- As a user, I want to rank my playlists
- The session should persist across refreshes

## Database Schema

* **`songs` Table**:
    * `isrc`: string
    * `genres`: string

## Acceptance Criteria
- [x] Users can paste a Spotify URL
- [ ] Performance under 3 seconds
