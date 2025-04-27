#!/usr/bin/env python3
# ─────────────────────────────────────────────────────────────────────────────
#  YT-DLP Made Easy – one-file GUI front-end for yt-dlp
#  Python 3.8+  |  customtkinter  |  plyer for desktop notifications
# ─────────────────────────────────────────────────────────────────────────────
import sys
import os
import subprocess
import threading
import traceback
import urllib.request
import json
import webbrowser
import gettext
from queue import Queue
from tkinter import filedialog, messagebox

import customtkinter as ctk
from plyer import notification

# ─────────────────────────────  i18n (stub)  ────────────────────────────────
_ = gettext.gettext  # later you can load .mo files

# ───────────────────────  global exception hook  ────────────────────────────
def _except_hook(exc_type, exc_value, exc_tb) -> None:
    tb = "".join(traceback.format_exception(exc_type, exc_value, exc_tb))
    print("UNHANDLED EXCEPTION:\n", tb, file=sys.stderr)
    try:
        messagebox.showerror(_("Fatal Error"), tb)
    except Exception:  # happens early before Tk exists
        pass
    sys.exit(1)

sys.excepthook = _except_hook

# ────────────────────────  theme + root window  ─────────────────────────────
ctk.set_appearance_mode("System")
ctk.set_default_color_theme("blue")

app = ctk.CTk()
app.title(_("YT-DLP Made Easy"))
app.geometry("860x760")
app.minsize(820, 700)
app.configure(padx=15, pady=15)

# ────────────────────────  paths & prefs  ───────────────────────────────────
if sys.platform.startswith("win"):
    base_dir = os.getenv("LOCALAPPDATA") or os.path.expanduser("~")
else:
    base_dir = os.getenv("XDG_DATA_HOME") or os.path.expanduser("~/.local/share")

cfg_dir        = os.path.join(base_dir, "yt-dlp-made-easy")
yt_dlp_exe     = os.path.join(cfg_dir,  "yt-dlp.exe")
log_file       = os.path.join(cfg_dir,  "activity.log")
prefs_file     = os.path.join(cfg_dir,  "prefs.json")
plugins_folder = os.path.join(cfg_dir,  "plugins")

os.makedirs(cfg_dir,        exist_ok=True)
os.makedirs(plugins_folder, exist_ok=True)

try:
    with open(prefs_file, "r", encoding="utf-8") as f:
        prefs: dict = json.load(f)
except Exception:
    prefs = {}

# ────────────────────────  globals  ─────────────────────────────────────────
current_process: subprocess.Popen | None = None
q: Queue = Queue()
pump_id: str | None = None
clipboard_last: str = ""

# ────────────────────────  helpers  ─────────────────────────────────────────
def get_expected_filename(url: str, folder: str) -> str | None:
    """Ask yt-dlp what filename it would use (without downloading)."""
    try:
        out = subprocess.run(
            [yt_dlp_exe, "--no-print-traffic", "--print", "filename", url],
            stdout=subprocess.PIPE,
            stderr=subprocess.DEVNULL,
            check=True,
            text=True
        )
        return os.path.join(folder, out.stdout.strip())
    except Exception:
        return None


def ask_overwrite(path: str) -> bool:
    return messagebox.askyesno(
        _("File Exists"),
        _("The file already exists:\n\n{0}\n\nReplace it?").format(path),
        parent=app
    )

# ────────────────────  worker & queue pump  ─────────────────────────────────
def _run_yt_dlp(args: list[str], post_hook: str | None = None) -> None:
    global current_process
    try:
        with open(log_file, "a", encoding="utf-8") as lg:
            current_process = subprocess.Popen(
                args, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True
            )
            for line in current_process.stdout:  # type: ignore
                q.put(line)
                lg.write(line)
            current_process.wait()
    except Exception as e:
        q.put(f"[ERROR] {e}\n")
    finally:
        current_process = None
        if post_hook:
            try:
                subprocess.Popen(post_hook, shell=True)
            except Exception:
                pass
        # desktop notification
        try:
            notification.notify(
                title=_("Download Complete"),
                message=args[1] if len(args) > 1 else _("Finished")
            )
        except Exception:
            pass


def threaded_yt_dlp(args: list[str], post_hook: str | None = None) -> None:
    threading.Thread(target=_run_yt_dlp, args=(args, post_hook),
                     daemon=True).start()


def pump_queue() -> None:
    while not q.empty():
        line = q.get()
        output_box.configure(state="normal")
        output_box.insert("end", line)
        output_box.see("end")
        output_box.configure(state="disabled")
    global pump_id
    pump_id = app.after(100, pump_queue)

# ───────────────────────  clipboard watcher  ────────────────────────────────
def poll_clipboard() -> None:
    global clipboard_last
    try:
        clip = app.clipboard_get()
        if clip.startswith("http") and clip != clipboard_last:
            url_text.delete("1.0", "end")
            url_text.insert("end", clip)
            clipboard_last = clip
    except Exception:
        pass
    app.after(2000, poll_clipboard)

# ─────────────────  ensure yt-dlp.exe is present  ───────────────────────────
def ensure_yt_dlp() -> None:
    if os.path.exists(yt_dlp_exe):
        return
    try:
        urllib.request.urlretrieve(
            "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp.exe",
            yt_dlp_exe
        )
    except Exception as e:
        messagebox.showerror(_("Download Failed"), str(e), parent=app)

# ─────────────────  GUI auto-update stub  ───────────────────────────────────
def check_app_update() -> None:
    # TODO: call GitHub API and offer an update
    pass

# ──────────────────────────  callbacks  ─────────────────────────────────────
def pick_folder() -> None:
    path = filedialog.askdirectory(parent=app)
    if path:
        folder_entry.delete(0, "end")
        folder_entry.insert(0, path)


def start_download() -> None:
    urls   = url_text.get("1.0", "end").strip().splitlines()
    folder = folder_entry.get().strip() or os.getcwd()

    for url in urls:
        if not url:
            continue

        expected = get_expected_filename(url, folder)
        if expected and os.path.exists(expected):
            if not ask_overwrite(expected):
                q.put(f"Skipped {url}\n")
                continue

        args = [yt_dlp_exe, url, "-P", folder]

        # audio vs video
        if audio_var.get():
            args += ["-x", "--audio-format", "mp3"]
        else:
            qstr = quality_var.get()
            if qstr != "Best":
                res = qstr.rstrip("p")
                args += ["-f", f"bestvideo[height<={res}]+bestaudio/best"]
            else:
                args += ["-f", "bv*+ba/best"]

        # subs & sponsorblock
        sub_lang = subs_var.get()
        if sub_lang:
            args += ["--write-subs", f"--sub-lang={sub_lang}"]
        if sponsor_var.get():
            args += ["--sponsorblock-remove", "all"]

        # rate limit & proxy
        rate = rate_entry.get().strip()
        proxy = proxy_entry.get().strip()
        if rate:
            args += ["--limit-rate", rate]
        if proxy:
            args += ["--proxy", proxy]

        # rename template
        tpl = rename_entry.get().strip()
        if tpl:
            args += ["-o", tpl]

        threaded_yt_dlp(args)

# ───────────────────────  UI construction  ─────────────────────────────────
tabs = ctk.CTkTabview(app, width=800, height=640)
tabs.pack(expand=True, fill="both")
tabs.add(_("Download"))
tabs.add(_("Log"))

dl_tab  = tabs.tab(_("Download"))
log_tab = tabs.tab(_("Log"))

# URL input frame ------------------------------------------------------------
url_frame = ctk.CTkFrame(dl_tab, corner_radius=8)
url_frame.pack(fill="x", pady=(0, 10))

ctk.CTkLabel(url_frame,
             text=_("Enter YouTube URLs (one per line):"),
             font=("Arial", 14)).pack(anchor="w", padx=10, pady=(10, 0))

url_text = ctk.CTkTextbox(url_frame, height=100, width=760)
url_text.pack(padx=10, pady=(5, 10))

# Settings grid --------------------------------------------------------------
settings = ctk.CTkFrame(dl_tab, corner_radius=8)
settings.pack(fill="x", pady=(0, 10))
for i in range(4):
    settings.grid_columnconfigure(i, weight=1, uniform="col")

# row 0 – folder
ctk.CTkLabel(settings, text=_("Save To:"), anchor="w").grid(row=0, column=0,
                                                           padx=10, pady=8,
                                                           sticky="w")
folder_entry = ctk.CTkEntry(settings)
folder_entry.grid(row=0, column=1, columnspan=2, padx=5, pady=8, sticky="ew")

browse_btn = ctk.CTkButton(settings, text=_("Browse"), width=80,
                           command=pick_folder)
browse_btn.grid(row=0, column=3, padx=10, pady=8)

# row 1 – audio / subs / sponsorblock
audio_var   = ctk.BooleanVar()
subs_var    = ctk.StringVar(value="")
sponsor_var = ctk.BooleanVar()

ctk.CTkCheckBox(settings, text=_("Audio Only"), variable=audio_var) \
    .grid(row=1, column=0, padx=10, sticky="w")

ctk.CTkOptionMenu(settings, values=["", "en", "es", "fr"],
                  variable=subs_var, dynamic_resizing=False) \
    .grid(row=1, column=1, sticky="ew")

ctk.CTkCheckBox(settings, text=_("Skip SponsorBlock"),
                variable=sponsor_var) \
    .grid(row=1, column=2, sticky="w")

# row 2 – quality / rate / proxy
quality_var = ctk.StringVar(value="Best")

ctk.CTkLabel(settings, text=_("Quality:"), anchor="w") \
    .grid(row=2, column=0, padx=10, pady=8, sticky="w")

ctk.CTkOptionMenu(settings, values=["Best", "1080p", "720p", "480p"],
                  variable=quality_var) \
    .grid(row=2, column=1, sticky="ew")

rate_entry  = ctk.CTkEntry(settings, placeholder_text=_("Rate e.g. 500K"))
rate_entry.grid(row=2, column=2, padx=5, sticky="ew")

proxy_entry = ctk.CTkEntry(settings, placeholder_text=_("Proxy URL"))
proxy_entry.grid(row=2, column=3, padx=10, sticky="ew")

# row 3 – rename template & presets
rename_entry = ctk.CTkEntry(settings,
                            placeholder_text=_("%(title)s.%(ext)s"))
rename_entry.grid(row=3, column=0, columnspan=2,
                  padx=10, pady=8, sticky="ew")

preset_var = ctk.StringVar(value="Default")
preset_menu = ctk.CTkOptionMenu(settings,
                                values=list(prefs.get("presets", {}).keys())
                                or ["Default"],
                                variable=preset_var)
preset_menu.grid(row=3, column=2, sticky="ew")

save_preset_btn = ctk.CTkButton(settings, text=_("Save Preset"), width=100)
save_preset_btn.grid(row=3, column=3, padx=10, sticky="ew")

# action buttons -------------------------------------------------------------
action_frame = ctk.CTkFrame(dl_tab, fg_color="transparent")
action_frame.pack(pady=10)

download_btn = ctk.CTkButton(action_frame, text=_("Download"), width=120,
                             command=start_download)
download_btn.grid(row=0, column=0, padx=10)

update_btn = ctk.CTkButton(action_frame, text=_("Update yt-dlp"), width=120,
                           command=lambda: threaded_yt_dlp([yt_dlp_exe, "-U"]))
update_btn.grid(row=0, column=1, padx=10)

open_folder_btn = ctk.CTkButton(action_frame, text=_("Open Folder"), width=120,
                                command=lambda:
                                webbrowser.open(f"file://{folder_entry.get()}"))
open_folder_btn.grid(row=0, column=2, padx=10)

# log tab --------------------------------------------------------------------
log_frame = ctk.CTkFrame(log_tab, corner_radius=8)
log_frame.pack(fill="both", expand=True, padx=10, pady=10)

output_box = ctk.CTkTextbox(log_frame, width=760, height=420)
output_box.pack(fill="both", expand=True, padx=10, pady=10)
output_box.configure(state="disabled")

ctk.CTkButton(log_frame, text=_("Open Log"),
              command=lambda: webbrowser.open(f"file://{log_file}")) \
    .pack(pady=(0, 10))

# ───────────────────────  run  ──────────────────────────────────────────────
app.after(100, pump_queue)
app.after(2000, poll_clipboard)
app.after_idle(ensure_yt_dlp)
app.after_idle(check_app_update)

print(">>> Entering mainloop()")
app.mainloop()
