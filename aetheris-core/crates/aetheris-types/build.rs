use std::io::Result;

fn main() -> Result<()> {
    // Proto files are at workspace root: aetheris-core/proto/domain/
    // Build script CWD is this crate's directory: aetheris-core/crates/aetheris-types/
    let proto_dir = "../../proto";

    prost_build::compile_protos(
        &[
            "../../proto/domain/job_event.proto",
            "../../proto/domain/agent_state.proto",
            "../../proto/domain/tool_call.proto",
            "../../proto/domain/checkpoint.proto",
            "../../proto/domain/memory.proto",
            "../../proto/domain/job_commands.proto",
        ],
        &[proto_dir],
    )?;
    Ok(())
}
