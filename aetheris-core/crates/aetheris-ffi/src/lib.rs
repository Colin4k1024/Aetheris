//! Aetheris FFI crate.
//!
//! Exposes C ABI functions for Go to call via CGo.
//! Uses cbindgen to generate C header files.

use std::ffi::{CStr, CString};
use std::os::raw::c_char;
use std::ptr;

// ── Error codes (must match AetherisError in aetheris-types) ────

const SUCCESS: i32 = 0;
const ERR_INVALID_INPUT: i32 = -1;
const ERR_VERSION_CONFLICT: i32 = -2;
const ERR_DATABASE: i32 = -3;
const ERR_TIMEOUT: i32 = -4;
const ERR_INTERNAL: i32 = -5;

// ── Memory management ──────────────────────────────────────────

/// Free a buffer allocated by Rust. Must be called by Go for every
/// `uint8_t*` returned by FFI functions.
///
/// # Safety
/// `ptr` must have been allocated by Rust (via `Box::into_raw` or similar).
#[no_mangle]
pub unsafe extern "C" fn aetheris_free(ptr: *mut u8, len: usize) {
    if !ptr.is_null() && len > 0 {
        let _ = Vec::from_raw_parts(ptr, len, len);
        // Vec is dropped here, freeing the memory
    }
}

/// Free a C string allocated by Rust.
///
/// # Safety
/// `ptr` must have been allocated by `CString::into_raw`.
#[no_mangle]
pub unsafe extern "C" fn aetheris_free_string(ptr: *mut c_char) {
    if !ptr.is_null() {
        let _ = CString::from_raw(ptr);
    }
}

// ── Helper: convert C string to Rust &str ──────────────────────

/// # Safety
/// `ptr` must be a valid, null-terminated C string.
unsafe fn cstr_to_str<'a>(ptr: *const c_char) -> Result<&'a str, i32> {
    if ptr.is_null() {
        return Err(ERR_INVALID_INPUT);
    }
    CStr::from_ptr(ptr)
        .to_str()
        .map_err(|_| ERR_INVALID_INPUT)
}

// ── JobStore FFI ───────────────────────────────────────────────

/// Opaque handle to a JobStore instance.
pub struct JobStoreHandle {
    // Will be populated in Phase 1 with actual store
    _dsn: String,
}

/// Create a new JobStore instance connected to the given DSN.
///
/// # Safety
/// `dsn` must be a valid null-terminated C string.
/// `out_handle` must be a valid pointer to a `uintptr_t`.
#[no_mangle]
pub unsafe extern "C" fn aetheris_jobstore_new(
    dsn: *const c_char,
    out_handle: *mut usize,
) -> i32 {
    let dsn_str = match cstr_to_str(dsn) {
        Ok(s) => s,
        Err(e) => return e,
    };

    if out_handle.is_null() {
        return ERR_INVALID_INPUT;
    }

    let handle = Box::new(JobStoreHandle {
        _dsn: dsn_str.to_string(),
    });

    *out_handle = Box::into_raw(handle) as usize;
    SUCCESS
}

/// Append an event to a job's event stream.
///
/// # Safety
/// All pointers must be valid. `event_json` must point to `event_len` bytes.
#[no_mangle]
pub unsafe extern "C" fn aetheris_jobstore_append(
    _handle: usize,
    job_id: *const c_char,
    expected_version: i32,
    event_json: *const u8,
    event_len: usize,
    out_new_version: *mut i32,
) -> i32 {
    let _job_id_str = match cstr_to_str(job_id) {
        Ok(s) => s,
        Err(e) => return e,
    };

    if event_json.is_null() || out_new_version.is_null() {
        return ERR_INVALID_INPUT;
    }

    // Phase 1: actual implementation will call oris kernel-postgres
    // For now, return stub
    let _event_slice = std::slice::from_raw_parts(event_json, event_len);
    let _expected = expected_version;

    *out_new_version = expected_version + 1;
    SUCCESS
}

/// Claim the next available job for a worker.
///
/// # Safety
/// `worker_id` must be a valid null-terminated C string.
/// `out_json` and `out_len` must be valid pointers.
#[no_mangle]
pub unsafe extern "C" fn aetheris_jobstore_claim(
    _handle: usize,
    worker_id: *const c_char,
    out_json: *mut *mut u8,
    out_len: *mut usize,
) -> i32 {
    let _worker_id_str = match cstr_to_str(worker_id) {
        Ok(s) => s,
        Err(e) => return e,
    };

    if out_json.is_null() || out_len.is_null() {
        return ERR_INVALID_INPUT;
    }

    // Phase 1: actual implementation
    // For now, return "no job" (empty response)
    *out_json = ptr::null_mut();
    *out_len = 0;
    SUCCESS
}

/// Destroy a JobStore handle.
///
/// # Safety
/// `handle` must be a valid handle returned by `aetheris_jobstore_new`.
#[no_mangle]
pub unsafe extern "C" fn aetheris_jobstore_free(handle: usize) {
    if handle != 0 {
        let _ = Box::from_raw(handle as *mut JobStoreHandle);
    }
}

// ── Executor FFI ───────────────────────────────────────────────

/// Opaque handle to an Executor instance.
pub struct ExecutorHandle {
    _jobstore_handle: usize,
}

/// Create a new Executor instance.
///
/// # Safety
/// `jobstore_handle` must be a valid handle from `aetheris_jobstore_new`.
/// `out_handle` must be a valid pointer.
#[no_mangle]
pub unsafe extern "C" fn aetheris_executor_new(
    jobstore_handle: usize,
    out_handle: *mut usize,
) -> i32 {
    if out_handle.is_null() {
        return ERR_INVALID_INPUT;
    }

    let handle = Box::new(ExecutorHandle {
        _jobstore_handle: jobstore_handle,
    });

    *out_handle = Box::into_raw(handle) as usize;
    SUCCESS
}

/// Run a single execution step.
///
/// # Safety
/// All pointers must be valid. `input_json` must point to `input_len` bytes.
#[no_mangle]
pub unsafe extern "C" fn aetheris_executor_run_step(
    _handle: usize,
    job_id: *const c_char,
    input_json: *const u8,
    input_len: usize,
    out_result: *mut *mut u8,
    out_result_len: *mut usize,
) -> i32 {
    let _job_id_str = match cstr_to_str(job_id) {
        Ok(s) => s,
        Err(e) => return e,
    };

    if input_json.is_null() || out_result.is_null() || out_result_len.is_null() {
        return ERR_INVALID_INPUT;
    }

    // Phase 2: actual implementation
    let _input_slice = std::slice::from_raw_parts(input_json, input_len);

    // Return stub success result
    let result = br#"{"type":"SUCCESS","output":null,"error_message":""}"#;
    let boxed = result.to_vec().into_boxed_slice();
    *out_result_len = boxed.len();
    *out_result = Box::into_raw(boxed) as *mut u8;
    SUCCESS
}

/// Destroy an Executor handle.
///
/// # Safety
/// `handle` must be a valid handle returned by `aetheris_executor_new`.
#[no_mangle]
pub unsafe extern "C" fn aetheris_executor_free(handle: usize) {
    if handle != 0 {
        let _ = Box::from_raw(handle as *mut ExecutorHandle);
    }
}

// ── Callback Registration (Rust → Go) ──────────────────────────

/// Callback function type for Go HTTP calls.
type GoHttpCallFn = extern "C" fn(
    url: *const c_char,
    body: *const u8,
    body_len: usize,
    out_resp: *mut *mut u8,
    out_resp_len: *mut usize,
) -> i32;

/// Callback function type for Go logging.
type GoLogFn = extern "C" fn(level: i32, message: *const c_char);

/// Global callback storage (set once at init).
static mut GO_HTTP_CALL: Option<GoHttpCallFn> = None;
static mut GO_LOG: Option<GoLogFn> = None;

/// Register Go callbacks for Rust to call back into Go.
///
/// # Safety
/// Function pointers must be valid for the lifetime of the process.
#[no_mangle]
pub unsafe extern "C" fn aetheris_register_callbacks(
    http_fn: GoHttpCallFn,
    log_fn: GoLogFn,
) {
    GO_HTTP_CALL = Some(http_fn);
    GO_LOG = Some(log_fn);
}

// ── Effects FFI ────────────────────────────────────────────────

/// Opaque handle to an EffectStore instance.
pub struct EffectStoreHandle {
    _placeholder: bool,
}

/// Create a new in-memory EffectStore instance.
///
/// # Safety
/// `out_handle` must be a valid pointer.
#[no_mangle]
pub unsafe extern "C" fn aetheris_effectstore_new(out_handle: *mut usize) -> i32 {
    if out_handle.is_null() {
        return ERR_INVALID_INPUT;
    }

    let handle = Box::new(EffectStoreHandle { _placeholder: true });
    *out_handle = Box::into_raw(handle) as usize;
    SUCCESS
}

/// Record a pending effect (Phase 1 of 2PC).
///
/// # Safety
/// All string pointers must be valid null-terminated C strings.
/// `input_json` must point to `input_len` bytes.
/// `out_effect_id` must be a valid pointer.
#[no_mangle]
pub unsafe extern "C" fn aetheris_effectstore_record_pending(
    _handle: usize,
    job_id: *const c_char,
    attempt_id: *const c_char,
    kind: *const c_char,
    input_json: *const u8,
    input_len: usize,
    idempotency_key: *const c_char,
    out_effect_id: *mut *mut c_char,
) -> i32 {
    let _job = match cstr_to_str(job_id) {
        Ok(s) => s,
        Err(e) => return e,
    };
    let _attempt = match cstr_to_str(attempt_id) {
        Ok(s) => s,
        Err(e) => return e,
    };
    let _kind_str = match cstr_to_str(kind) {
        Ok(s) => s,
        Err(e) => return e,
    };
    let _key = match cstr_to_str(idempotency_key) {
        Ok(s) => s,
        Err(e) => return e,
    };

    if input_json.is_null() || out_effect_id.is_null() {
        return ERR_INVALID_INPUT;
    }

    // Phase 1 actual implementation will use the EffectStore trait
    // For now, generate a placeholder UUID
    let id = "effect-placeholder";
    match CString::new(id) {
        Ok(c_str) => {
            *out_effect_id = c_str.into_raw();
            SUCCESS
        }
        Err(_) => ERR_INTERNAL,
    }
}

/// Confirm a pending effect (Phase 2 — commit).
///
/// # Safety
/// `effect_id` must be a valid null-terminated C string.
/// `output_json` must point to `output_len` bytes.
#[no_mangle]
pub unsafe extern "C" fn aetheris_effectstore_confirm(
    _handle: usize,
    effect_id: *const c_char,
    output_json: *const u8,
    output_len: usize,
) -> i32 {
    let _id = match cstr_to_str(effect_id) {
        Ok(s) => s,
        Err(e) => return e,
    };

    if output_json.is_null() {
        return ERR_INVALID_INPUT;
    }

    let _output = std::slice::from_raw_parts(output_json, output_len);
    SUCCESS
}

/// Rollback a pending effect (Phase 2 — abort).
///
/// # Safety
/// `effect_id` and `reason` must be valid null-terminated C strings.
#[no_mangle]
pub unsafe extern "C" fn aetheris_effectstore_rollback(
    _handle: usize,
    effect_id: *const c_char,
    reason: *const c_char,
) -> i32 {
    let _id = match cstr_to_str(effect_id) {
        Ok(s) => s,
        Err(e) => return e,
    };
    let _reason = match cstr_to_str(reason) {
        Ok(s) => s,
        Err(e) => return e,
    };
    SUCCESS
}

/// Destroy an EffectStore handle.
///
/// # Safety
/// `handle` must be a valid handle from `aetheris_effectstore_new`.
#[no_mangle]
pub unsafe extern "C" fn aetheris_effectstore_free(handle: usize) {
    if handle != 0 {
        let _ = Box::from_raw(handle as *mut EffectStoreHandle);
    }
}
