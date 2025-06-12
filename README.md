# go-queue 📨

> A minimal, pull-based message queue written in Go — built for learning and showcasing internals of messaging systems.

---

## 🌟 Features (so far)

- ✅ Topic-based architecture (produce/consume by topic)
- ✅ Message persistence using Write-Ahead Log (WAL)
- ✅ At-least-once delivery guarantee
- ✅ Pull-based consumption (consumer polls for messages)
- ✅ Message acknowledgment + retry on failure
- ✅ In-memory in-flight tracking
- ✅ Crash recovery from WAL

---

## 🔧 Upcoming Features

- [ ] Configurable timeouts and retry limits from config file
- [ ] Checkpointing + WAL log compaction
- [ ] Dead-letter queue support
- [ ] Segment-based WAL design

---

## 🚀 Why This Project?

I built `go-queue` from the ground up to deeply understand:

- How message queues like RabbitMQ or Kafka work internally
- Core concurrency, I/O, and persistence patterns in Go
- WAL design, retry mechanisms, delivery guarantees
- Building reliable systems without relying on external libraries

---

## 🔨 How It Works

- A **producer** publishes a message to a topic → persisted to WAL
- A **consumer** polls for new messages → delivered with ID
- Consumer **must ACK** within timeout or message is retried
- If the queue server crashes:

  - WAL is **replayed**
  - In-memory state is **reconstructed** (pending + in-flight)

---

## 📦 Getting Started

```bash
git clone https://github.com/yourusername/go-queue.git
cd go-queue
go run main.go
```

---

## 🧰 Inspiration

- Apache Kafka
- RabbitMQ
- Log-structured storage systems

---

## 🙌 Contributions Welcome

This project is actively being extended!
Feel free to open issues, ask questions, or fork it for your own learning journey.
