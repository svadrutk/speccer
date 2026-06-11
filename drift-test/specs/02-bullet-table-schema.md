# Bullet List Schema Test

## Database Schema Changes

* **`sessions` Table**:
    * `session_id`: UUID4
    * `playlist_id`: string (nullable for legacy sessions)
    * `user_id`: UUID4 (nullable)
    * `playlist_name`: string
    * `source_platform`: string

* **`songs` Table**:
    * `isrc`: string
    * `artist_name`: string
    * `genres`: string

* **`users` Table**:
    * `email`: string
    * `display_name`: string
