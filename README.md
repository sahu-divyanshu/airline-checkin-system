# PostgreSQL Concurrency and Row-Level Locks

This repository demonstrates a physical stress test of database transaction isolation in a high-demand booking system using Go and PostgreSQL.

## The Objective
Force a database race condition with concurrent workers accessing a shared resource, then systematically resolve the data corruption and throughput bottlenecks using row-level locks.

## Execution Data

### Approach 2: The Race Condition (Standard SELECT)
* **Mechanism:** Fired concurrent workers at a shared resource using a standard `SELECT` followed by an `UPDATE`.
* **Result:** Total system corruption. Multiple concurrent workers bypassed the read check simultaneously, resulting in double-booked states. Data integrity failed completely.

### Approach 3: The Bottleneck (Pessimistic Lock)
* **Mechanism:** Implemented `SELECT ... FOR UPDATE`.
* **Result:** Data integrity was restored. The database engine successfully serialized the concurrent requests. 
* **Performance Cost:** Throughput collapsed. The strict serialization bottlenecked the system, resulting in an 800ms execution time.

### Approach 4: High-Throughput Locking (`SKIP LOCKED`)
* **Mechanism:** Upgraded the query to `SELECT ... FOR UPDATE SKIP LOCKED`. 
* **Result:** Absolute data integrity maintained. Workers dynamically ignored locked rows and grabbed the next available record in parallel.
* **Performance Gain:** Execution time dropped by 50 percent to 400ms. 

## Implementation Details
The codebase contains three isolated Go implementations to reproduce these exact scenarios.
