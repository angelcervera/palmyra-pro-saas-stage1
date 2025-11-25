# UI demo
You are the front end developer using the persistence-sdk at packages/persistence-sdk
You are going to write a simple app to prove how it works.
It MUST be a new app, separate from web-admin. For example. `apps/demo`

This demo is going to use the persistence-sdk to store the data locally.
The domain name that we are going to use is as example is going to be called `Person`

`persistence-sdk at packages/persistence-sdk` is still in development.
Every domain (currently only `Person`) is going to have a `.ts` file that will be a wrapper around the persistence-sdk.

## Entity format

The schema that we are going to use is called `Person`
This is the json schema for the example entity we are going to use:
```json

{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "Person",
  "type": "object",
  "additionalProperties": false,
  "required": ["name", "surname", "age", "dob", "phoneNumber", "photo"],
  "properties": {
    "name": {
      "type": "string",
      "minLength": 1,
      "description": "Given name"
    },
    "surname": {
      "type": "string",
      "minLength": 1,
      "description": "Family name"
    },
    "age": {
      "type": "integer",
      "minimum": 0,
      "maximum": 150,
      "description": "Age in whole years"
    },
    "dob": {
      "type": "string",
      "format": "date",
      "description": "Date of birth (YYYY-MM-DD)"
    },
    "phoneNumber": {
      "type": "string",
      "pattern": "^\\+?[1-9]\\d{7,14}$",
      "description": "E.164-style phone number"
    },
    "photo": {
      "type": "string",
      "format": "uri",
      "description": "URL to profile photo (HTTPS recommended)"
    }
  }
}
```

- Fields use camelCase and basic validation (non-empty strings, sensible age bounds, E.164 phone pattern, date format, URI for photo).
- All entity records, including the `Person` entity, are wrapped around an object with metadata. This is the tS representation:
  ```typescript
      export interface EntityWithSyncMeta<TPayload> {
      queuedForSync: boolean;         // boolean indicating if the entity is queued for sync.
      lastSynced: Date | null;        // timestamp of the last sync. null if never synced.
      lastSyncError: string | null;   // error message of the last sync.
      entityId: string;               // uuid entity identifier
      entityVersion: string;          // semantic version e.g. "1.0.0"
      entitySchemaId: string;         // uuid schema identifier
      entitySchemaVersion: string;    // schema version e.g. "1.0.0"
      entity: TPayload;               // the actual entity payload
      }
  ```

## UI flow

- The user can create, update, delete and synchronize entities.
- The first page is a list view of all entities.

### List view
- The view lists all entries.
- The list is a table.
- Every row in the table has the columns `name`, `surname`, `age`, `dob`, `phoneNumber` and `photo`.
- The photo will be the second column, and wil ve a small image with the photo.
- The first column will be a checkbox to select the row.
- The first column in the header of the table will be a checkbox to select all rows.
- The table will have pagination.
- The view will have a checkbox to filter by `queuedForSync`.
- The view will have a button to synchronize all entities.
- The view will have a button to create a new entity.
- Every row will have a button to edit the entity that will redirect to the Update view.

### Create view
- It will have a form with the fields from the schema.
- It will have a button to submit the form.
- It will redirect to the List view after successful submission.
- It will show errors if any.
- It will have a button to cancel the form.

### Update view
- It will be like the Create view, but with the entity data pre-filled.
- It will have a button to submit the form.



## Technologies.
In the demo we will use:
- React
- Typescript
- ShadCN
- Before selecting other technologies, look at apps/web-admin
