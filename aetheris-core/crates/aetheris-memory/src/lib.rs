//! Aetheris memory crate.
//!
//! 4-tier memory system for agents:
//! - **Short-term**: Current session scope (in-memory, ephemeral)
//! - **Working**: Current job scope (in-memory, scoped to execution)
//! - **Episodic**: Session/job summaries (persistent, append-only)
//! - **Long-term**: Durable across sessions (persistent, searchable)

mod error;
mod episodic;
mod longterm;
mod memory;
mod shortterm;
mod store;
mod types;
mod working;

pub use error::MemoryError;
pub use episodic::InMemoryEpisodicStore;
pub use longterm::InMemoryLongTermStore;
pub use memory::MemoryManager;
pub use shortterm::ShortTermMemory;
pub use store::MemoryStore;
pub use types::*;
pub use working::WorkingMemory;
