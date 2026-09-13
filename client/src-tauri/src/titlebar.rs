//! Windows 原生标题栏着色：用 DWM caption/border 覆盖系统强调色。
//! 非 Windows 或旧系统（Win10 / 不支持的 attribute）一律静默忽略。

#[cfg(windows)]
mod dwm {
  use std::ffi::c_void;

  /// `DWMWA_BORDER_COLOR` — Windows 11 Build 22000+
  pub const DWMWA_BORDER_COLOR: u32 = 34;
  /// `DWMWA_CAPTION_COLOR` — Windows 11 Build 22000+
  pub const DWMWA_CAPTION_COLOR: u32 = 35;

  #[link(name = "dwmapi")]
  extern "system" {
    pub fn DwmSetWindowAttribute(
      hwnd: *mut c_void,
      dw_attribute: u32,
      pv_attribute: *const c_void,
      cb_attribute: u32,
    ) -> i32;
  }
}

/// 设置当前窗口标题栏 / 边框颜色。
/// `caption` / `border` 为 COLORREF（`0x00BBGGRR`）。
#[tauri::command]
pub fn set_titlebar_colors(window: tauri::WebviewWindow, caption: u32, border: u32) {
  #[cfg(windows)]
  {
    apply_dwm_colors(&window, caption, border);
  }
  #[cfg(not(windows))]
  {
    let _ = (window, caption, border);
  }
}

#[cfg(windows)]
fn apply_dwm_colors(window: &tauri::WebviewWindow, caption: u32, border: u32) {
  let Ok(hwnd) = window.hwnd() else {
    return;
  };
  // tauri 的 HWND 是 `windows::Win32::Foundation::HWND(*mut c_void)`
  let hwnd = hwnd.0;
  unsafe {
    set_color(hwnd, dwm::DWMWA_CAPTION_COLOR, caption);
    set_color(hwnd, dwm::DWMWA_BORDER_COLOR, border);
  }
}

#[cfg(windows)]
unsafe fn set_color(hwnd: *mut std::ffi::c_void, attribute: u32, color: u32) {
  // Win10 / 旧 build 不支持 34/35，HRESULT 非 0 也直接忽略
  let _ = dwm::DwmSetWindowAttribute(
    hwnd,
    attribute,
    (&color as *const u32).cast(),
    std::mem::size_of::<u32>() as u32,
  );
}
