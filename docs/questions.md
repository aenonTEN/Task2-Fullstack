Project Clarity Decision Log

1 - Who owns a Client Data Scopes
The Confusion- The prompt talks about high level Institutions but also Clients and Suppliers It wasnt clear if a Client is a big organization itself or just a small shop we do business with at a local branch
The Real World Interpretation- We assumed that a Client or Supplier is an outside partner that a specific branch Institution works with For example the Downtown Pharmacy Institution has its own list of Local Patients Clients
The Fix- We linked every Client Supplier to a specific institution id This keeps things private a branch in one city wont see the patient list of a branch in another city preventing data leaks

2 - Handling Expirations without a Clock Auto Deactivation
The Confusion- The system has to automatically deactivate expired licenses but since its an offline intranet app there isnt always a fancy cloud service running in the background to watch the clock 24 7
The Real World Interpretation- We cant just hope a background task runs we need to guarantee that no one can trade with an expired supplier the moment their license dies
The Fix- We implemented Check on Arrival logic Every time someone logs in or tries to create an order the system silently checks the dates If its expired it flips the status to Inactive right then and there Its reliable and doesnt require complex server scheduling

3 - The Wait Your Turn Rule 7 Day Purchase Limit
The Confusion- Once every 7 days is a bit vague Does the limit reset every Monday morning or is it a strict timer
The Real World Interpretation In pharma safety isnt about the calendar its about the time elapsed since the last dose or purchase We should prevent someone from buying on Sunday night and then again on Monday morning
The Fix- We set a strict 168 hour rolling window If you bought a prescription at 2 00 PM on Tuesday the system wont let you buy it again until 2 01 PM the following Tuesday Its the safest way to prevent medication hoarding

4 - Why we dont truly Delete anything Non Repudiation
The Confusion- The prompt mentions deleting records but also says the Audit Log must be non modifiable and show before after values You cant show a before value if the record is gone from the database
The Real World Interpretation- In a regulated industry deleting is a liability If a recruiter accidentally deletes a candidate we still need to know who they were for legal reasons
The Fix- We banned Hard Deletes When a user hits Delete we just hide the record from view Soft Delete This keeps the database clean for the user but preserves the history for the auditors Its the Paper Trail approach

5 - Making the Match Score make sense Intelligent Search
The Confusion - The prompt asks for a 0 100 score but doesnt say how to calculate it A score without an explanation is frustrating for a recruiter
The Real World Interpretation- A recruiter needs to trust the system If it says 80 they need to know why so they can explain it to their manager
The Fix- We created a Points System that mimics human logic 50 points for the right skills 30 for the right years of experience and 20 for education We then speak the results e g Great skills match but missing the preferred Masters degree making the AI feel more like a helpful assistant and less like a black box

6 - Time Zone And Clock Authority For 168 Hour Rules
The Confusion- The 168 hour rolling limit depends on time calculation but the prompt does not define if we use local machine time or a standardized source
The Real World Interpretation- In a distributed branch setup daylight saving and local machine drift can create inconsistent restriction decisions
The Fix- We use server authoritative UTC timestamps for all policy windows and persist both UTC and display-local converted times for audit readability

7 - Daylight Saving Time Edge Cases
The Confusion- A seven day rule can behave differently across DST changes if interpreted as calendar dates rather than elapsed hours
The Real World Interpretation- Safety restrictions should not weaken or tighten accidentally due to clock shifts
The Fix- Restriction checks use exact elapsed duration in seconds equivalent to 168 hours and never calendar-day boundary logic

8 - Reactivation Policy After Qualification Renewal
The Confusion- Prompt defines auto deactivation but does not define if reactivation should be manual or automatic after valid renewal documents are uploaded
The Real World Interpretation- Operations teams need predictable behavior to avoid blocked transactions after compliance updates
The Fix- Reactivation is automatic only after successful document validation and qualification approval workflow completion and every transition is audit logged

9 - Duplicate Merge Conflict Precedence
The Confusion- Duplicate candidate merge is required but there is no rule for conflicting values between records for critical fields
The Real World Interpretation- Recruiters need deterministic outcomes so they can trust data lineage and rollback decisions
The Fix- We enforce field precedence rules primary record wins by default configurable overrides are allowed for selected fields and all overridden fields are tracked in merge audit details

10 - Case Number Serial Contention Under Concurrency
The Confusion- Unique case numbering is specified but not the behavior under high concurrent submissions within the same institution and date
The Real World Interpretation- Parallel submissions can create duplicate numbers or failed writes if allocation is not atomic
The Fix- Serial allocation is atomic per institution per date using transactional lock or sequence table with retry policy to guarantee uniqueness without collisions
