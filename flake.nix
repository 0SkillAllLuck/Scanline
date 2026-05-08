{

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        # Workaround for libfyaml-0.9.6 in nixpkgs: the shipped libfyaml.pc
        # contains a stray "none required" token in Libs: (an autoconf
        # AC_SEARCH_LIBS artifact leaked from libfyaml.pc.in). Every consumer
        # that runs `pkg-config --libs libfyaml` inherits it, and on Darwin
        # clang treats "none" and "required" as missing input files, so any
        # downstream build (e.g. appstream, transitively libadwaita) fails to
        # link. Strip the bad token from the .pc file at fixup time.
        libfyamlPcFix = final: prev: {
          libfyaml = prev.libfyaml.overrideAttrs (old: {
            postFixup = (old.postFixup or "") + ''
              for pc in "$dev"/lib/pkgconfig/*.pc; do
                substituteInPlace "$pc" --replace-quiet " none required " " "
              done
            '';
          });
        };
        pkgs = import nixpkgs {
          inherit system;
          overlays = [ libfyamlPcFix ];
        };
        libraryPath = pkgs.symlinkJoin {
          name = "scanline-puregotk-lib-folder";
          paths = with pkgs; [
            cairo
            gdk-pixbuf
            glib.out
            graphene
            pango.out
            gtk4
            libadwaita
            gobject-introspection
            librsvg
            libsecret
            adwaita-icon-theme
          ];
        };
        # nixpkgs librsvg on Darwin ships a broken loaders.cache (the SVG entry is
        # missing) and the loader dylib has no LC_RPATH, so dlopen of @rpath/librsvg-2.2.dylib
        # fails. Provide our own cache; pair with DYLD_FALLBACK_LIBRARY_PATH below.
        # The trailing blank line is required: gdk-pixbuf parses entries terminated by \n\n.
        darwinPixbufLoadersCache = pkgs.writeText "scanline-pixbuf-loaders.cache" ''
          # GdkPixbuf Image Loader Modules file - Scanline devShell override
          "${pkgs.librsvg.out}/lib/gdk-pixbuf-2.0/2.10.0/loaders/libpixbufloader_svg.dylib"
          "svg" 6 "gdk-pixbuf" "Scalable Vector Graphics" "LGPL"
          "image/svg+xml" "image/svg" "image/svg-xml" "image/vnd.adobe.svg+xml" "text/xml-svg" "image/svg+xml-compressed" ""
          "svg" "svgz" "svg.gz" ""
          " <svg" "*    " 100
          " <!DOCTYPE svg" "*             " 100

        '';
        version = "0.4.0";
        # Builds a self-contained Scanline.app for distribution outside Nix.
        # The derivation copies the runtime dylib closure into Contents/Frameworks,
        # rewrites every /nix/store LC_LOAD_DYLIB to @rpath/<basename>, and adds
        # an LC_RPATH so the bundle resolves its own libraries from
        # @executable_path/../Frameworks. The wrapper at Contents/MacOS/Scanline
        # only needs to set the non-dylib runtime knobs (gst plugins, gio
        # modules, gdk-pixbuf loader cache, schemas, icon themes).
        scanlineDarwinApp = pkgs.runCommand "scanline-darwin-app"
          {
            inherit version;
            src = pkgs.lib.cleanSource ./.;
            inherit (self.packages.${system}) scanline;
            nativeBuildInputs = with pkgs; [
              darwin.cctools
              libicns
              librsvg
              glib.dev
              gdk-pixbuf.dev
            ];
            runtimeLibs = with pkgs; [
              gtk4
              libadwaita
              glib.out
              cairo
              gdk-pixbuf
              graphene
              pango.out
              libsecret
              librsvg.out
              appstream
            ];
            gstPlugins = with pkgs.gst_all_1; [
              gstreamer
              gst-plugins-base
              gst-plugins-good
              gst-plugins-bad
              gst-plugins-ugly
              gst-plugins-rs
              gst-libav
            ];
            glibNetworking = pkgs.glib-networking;
            adwaitaIcons = pkgs.adwaita-icon-theme;
            hicolorIcons = pkgs.hicolor-icon-theme;
            librsvgPkg = pkgs.librsvg;
          }
          ''
            set -e
            APP="$out/Scanline.app"
            mkdir -p "$APP/Contents/MacOS"
            mkdir -p "$APP/Contents/Resources/glib-2.0/schemas"
            mkdir -p "$APP/Contents/Resources/share/icons"
            mkdir -p "$APP/Contents/Frameworks/gstreamer-1.0"
            mkdir -p "$APP/Contents/Frameworks/gio/modules"
            mkdir -p "$APP/Contents/Frameworks/gdk-pixbuf-2.0/2.10.0/loaders"

            # The closure has two libiconv.2.dylib files with incompatible ABIs:
            # Apple's (no `_libiconv` symbol; gettext links against this) and GNU's
            # (exports `_libiconv`; libidn2 + transitively libgnutls/libcurl/libpsl
            # link against this). Flattening both to the same basename in Frameworks
            # breaks whichever one loses the dedup race. Rename the GNU variant to
            # libiconv-gnu.2.dylib so both can coexist.
            dest_basename_for() {
              local src="$1"
              local base
              base=$(basename "$src")
              case "$base" in
                libiconv.2.dylib|libiconv.dylib)
                  if nm "$src" 2>/dev/null | grep -q "T _libiconv$"; then
                    echo "libiconv-gnu.2.dylib"
                    return
                  fi
                  ;;
              esac
              echo "$base"
            }

            # Copy a dylib + recursively bundle its /nix/store deps to Contents/Frameworks/.
            # $1 = source dylib path, $2 = dest dir for THIS dylib (deps always go to Frameworks/).
            copy_and_fix_dylib() {
              local src="$1"
              local dest_dir="$2"
              local base
              base=$(dest_basename_for "$src")
              local dest="$dest_dir/$base"
              [ -e "$dest" ] && return 0
              cp -L "$src" "$dest"
              chmod +w "$dest"
              install_name_tool -id "@rpath/$base" "$dest" 2>/dev/null || true
              install_name_tool -add_rpath "@executable_path/../Frameworks" "$dest" 2>/dev/null || true
              # Strip any /nix/store-prefixed LC_RPATHs the dylib carries; dyld searches
              # rpaths in order, so a leftover /nix/store path would resolve before our
              # @executable_path/../Frameworks and load from /nix/store on dev machines
              # (and silently fail on user machines).
              local rp
              otool -l "$dest" | awk '/cmd LC_RPATH/{f=1} f && /path /{print $2; f=0}' | while IFS= read -r rp; do
                case "$rp" in
                  /nix/store/*)
                    install_name_tool -delete_rpath "$rp" "$dest" 2>/dev/null || true
                    ;;
                esac
              done
              local dep dep_base
              while IFS= read -r dep; do
                [ -z "$dep" ] && continue
                case "$dep" in
                  /nix/store/*)
                    dep_base=$(dest_basename_for "$dep")
                    copy_and_fix_dylib "$dep" "$APP/Contents/Frameworks"
                    install_name_tool -change "$dep" "@rpath/$dep_base" "$dest" 2>/dev/null || true
                    ;;
                esac
              done < <(otool -L "$dest" | tail -n +2 | awk 'NF>0 {print $1}')
            }

            # 1. Copy and fix the main binary.
            # nixpkgs's wrapGAppsHook4 + wrapProgram chain produces a stack of
            # tiny wrappers that exec the next, ending in the real Go binary at
            # `..scanline-wrapped-wrapped`. The wrappers re-set DYLD env vars to
            # /nix/store paths, so bundling the outer wrapper gives a non-relocatable
            # bundle that only runs on machines where /nix/store exists. Skip the
            # wrappers and bundle the unwrapped binary directly.
            SOURCE_BIN="$scanline/bin/..scanline-wrapped-wrapped"
            if [ ! -f "$SOURCE_BIN" ]; then
              echo "Error: expected unwrapped binary at $SOURCE_BIN" >&2
              echo "wrapGAppsHook4 wrapping convention may have changed; contents of $scanline/bin:" >&2
              ls -la "$scanline/bin/" >&2
              exit 1
            fi
            cp "$SOURCE_BIN" "$APP/Contents/MacOS/scanline-bin"
            chmod +w "$APP/Contents/MacOS/scanline-bin"
            install_name_tool -add_rpath "@executable_path/../Frameworks" \
              "$APP/Contents/MacOS/scanline-bin" 2>/dev/null || true
            # Strip /nix/store-prefixed LC_RPATHs (e.g. gstreamer's lib dir baked in
            # by buildGoModule), so dyld doesn't resolve @rpath against /nix/store first.
            otool -l "$APP/Contents/MacOS/scanline-bin" \
              | awk '/cmd LC_RPATH/{f=1} f && /path /{print $2; f=0}' \
              | while IFS= read -r rp; do
                  case "$rp" in
                    /nix/store/*)
                      install_name_tool -delete_rpath "$rp" \
                        "$APP/Contents/MacOS/scanline-bin" 2>/dev/null || true
                      ;;
                  esac
                done
            while IFS= read -r dep; do
              [ -z "$dep" ] && continue
              case "$dep" in
                /nix/store/*)
                  dep_base=$(dest_basename_for "$dep")
                  copy_and_fix_dylib "$dep" "$APP/Contents/Frameworks"
                  install_name_tool -change "$dep" "@rpath/$dep_base" \
                    "$APP/Contents/MacOS/scanline-bin" 2>/dev/null || true
                  ;;
              esac
            done < <(otool -L "$APP/Contents/MacOS/scanline-bin" | tail -n +2 | awk 'NF>0 {print $1}')

            # 2. Bundle puregotk-dlopened libs (not in the binary's LC_LOAD_DYLIB)
            for pkg in $runtimeLibs; do
              if [ -d "$pkg/lib" ]; then
                for dylib in "$pkg/lib"/*.dylib; do
                  [ -f "$dylib" ] || continue
                  copy_and_fix_dylib "$dylib" "$APP/Contents/Frameworks"
                done
              fi
            done

            # 3. GStreamer plugins
            for pkg in $gstPlugins; do
              if [ -d "$pkg/lib/gstreamer-1.0" ]; then
                for plugin in "$pkg/lib/gstreamer-1.0"/*.dylib; do
                  [ -f "$plugin" ] || continue
                  copy_and_fix_dylib "$plugin" "$APP/Contents/Frameworks/gstreamer-1.0"
                done
              fi
            done

            # 4. glib-networking GIO modules (TLS for libsoup). Note: glib uses
            # .so extensions for loadable modules even on Darwin.
            if [ -d "$glibNetworking/lib/gio/modules" ]; then
              for mod in "$glibNetworking/lib/gio/modules"/*.so "$glibNetworking/lib/gio/modules"/*.dylib; do
                [ -f "$mod" ] || continue
                copy_and_fix_dylib "$mod" "$APP/Contents/Frameworks/gio/modules"
              done
            fi

            # 5. librsvg's GdkPixbuf loader (for SVG icons), then generate the
            # loaders.cache template via gdk-pixbuf-query-loaders. The canonical tool
            # reads each loader's metadata via dlopen, so the cache always matches
            # what gdk-pixbuf actually expects. We replace the build-time .app path
            # with @APPDIR@; the wrapper sed-substitutes it at launch.
            for loader in "$librsvgPkg/lib/gdk-pixbuf-2.0/2.10.0/loaders"/*.dylib; do
              [ -f "$loader" ] || continue
              copy_and_fix_dylib "$loader" "$APP/Contents/Frameworks/gdk-pixbuf-2.0/2.10.0/loaders"
            done
            mkdir -p "$APP/Contents/Resources"
            # Our copied loader has LC_RPATH @executable_path/../Frameworks; that's
            # relative to scanline-bin at runtime but resolves to query-loaders' own
            # /nix/store path here. Point dyld at our bundled Frameworks dir so it
            # can resolve the loader's deps (including libiconv, which isn't in the
            # devshell libraryPath) and successfully dlopen the loader.
            DYLD_FALLBACK_LIBRARY_PATH="$APP/Contents/Frameworks" \
              gdk-pixbuf-query-loaders \
                "$APP/Contents/Frameworks/gdk-pixbuf-2.0/2.10.0/loaders"/*.dylib \
              | sed "s|$APP/Contents|@APPDIR@|g" \
              > "$APP/Contents/Resources/pixbuf-loaders.cache.in"

            # 6. GSettings schemas
            cp "$src/assets/meta/dev.skillless.Scanline.gschema.xml" \
              "$APP/Contents/Resources/glib-2.0/schemas/"
            glib-compile-schemas "$APP/Contents/Resources/glib-2.0/schemas/"

            # 7. Adwaita + hicolor icon themes
            cp -R "$adwaitaIcons/share/icons/Adwaita" "$APP/Contents/Resources/share/icons/"
            if [ -d "$hicolorIcons/share/icons/hicolor" ]; then
              cp -R "$hicolorIcons/share/icons/hicolor" "$APP/Contents/Resources/share/icons/"
            fi

            # 8. Scanline.icns from app.svg via rsvg-convert + png2icns
            mkdir -p icns-staging
            for sz in 16 32 64 128 256 512 1024; do
              rsvg-convert -w "$sz" -h "$sz" "$src/assets/icons/app.svg" \
                -o "icns-staging/icon_$sz.png"
            done
            png2icns "$APP/Contents/Resources/Scanline.icns" \
              icns-staging/icon_16.png icns-staging/icon_32.png \
              icns-staging/icon_64.png icns-staging/icon_128.png \
              icns-staging/icon_256.png icns-staging/icon_512.png \
              icns-staging/icon_1024.png

            # 9. Info.plist (uses unquoted heredoc so $version is expanded by the build shell)
            cat > "$APP/Contents/Info.plist" <<PLIST_EOF
            <?xml version="1.0" encoding="UTF-8"?>
            <!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
            <plist version="1.0">
            <dict>
                <key>CFBundleDevelopmentRegion</key>
                <string>en</string>
                <key>CFBundleDisplayName</key>
                <string>Scanline</string>
                <key>CFBundleExecutable</key>
                <string>Scanline</string>
                <key>CFBundleIconFile</key>
                <string>Scanline</string>
                <key>CFBundleIdentifier</key>
                <string>dev.skillless.Scanline</string>
                <key>CFBundleInfoDictionaryVersion</key>
                <string>6.0</string>
                <key>CFBundleName</key>
                <string>Scanline</string>
                <key>CFBundlePackageType</key>
                <string>APPL</string>
                <key>CFBundleShortVersionString</key>
                <string>$version</string>
                <key>CFBundleVersion</key>
                <string>$version</string>
                <key>LSMinimumSystemVersion</key>
                <string>13.0</string>
                <key>NSHighResolutionCapable</key>
                <true/>
                <key>CFBundleURLTypes</key>
                <array>
                    <dict>
                        <key>CFBundleURLName</key>
                        <string>Plex URL</string>
                        <key>CFBundleURLSchemes</key>
                        <array>
                            <string>plex</string>
                        </array>
                    </dict>
                </array>
            </dict>
            </plist>
            PLIST_EOF

            # 10. Wrapper launcher (CFBundleExecutable). Quoted heredoc — no expansion at build time.
            cat > "$APP/Contents/MacOS/Scanline" <<'WRAPPER_EOF'
            #!/bin/bash
            set -e
            APP_DIR="$(cd "$(dirname "$0")/.." && pwd -P)"
            APP_FW="$APP_DIR/Frameworks"
            APP_RES="$APP_DIR/Resources"

            # GdkPixbuf loaders.cache: substitute @APPDIR@ in the build-time template
            # (generated by gdk-pixbuf-query-loaders) with the actual install path.
            PIXBUF_CACHE="$(mktemp -t scanline-pixbuf-XXXXXX)"
            sed "s|@APPDIR@|$APP_DIR|g" "$APP_RES/pixbuf-loaders.cache.in" > "$PIXBUF_CACHE"

            # puregotk dlopens libgtk-4.1.dylib and friends by basename. dyld bare-name
            # search ignores LC_RPATH, so DYLD_FALLBACK_LIBRARY_PATH is what makes it
            # find our bundled libs. PUREGOTK_LIB_FOLDER is the puregotk-specific knob.
            export DYLD_FALLBACK_LIBRARY_PATH="$APP_FW"
            export PUREGOTK_LIB_FOLDER="$APP_FW"
            export GDK_PIXBUF_MODULE_FILE="$PIXBUF_CACHE"
            export GST_PLUGIN_PATH="$APP_FW/gstreamer-1.0"
            export GIO_EXTRA_MODULES="$APP_FW/gio/modules"
            export XDG_DATA_DIRS="$APP_RES/share"
            export GSETTINGS_SCHEMA_DIR="$APP_RES/glib-2.0/schemas"

            exec "$(dirname "$0")/scanline-bin" "$@"
            WRAPPER_EOF
            chmod +x "$APP/Contents/MacOS/Scanline"
          '';
      in
      pkgs.lib.recursiveUpdate
      {
        devShell = pkgs.mkShell ({
          PUREGOTK_LIB_FOLDER = "${libraryPath}/lib";
          GSETTINGS_SCHEMA_DIR = "./assets/meta";
          SCANLINE_DEBUG = "1";
          GST_PLUGIN_PATH = pkgs.lib.makeSearchPath "lib/gstreamer-1.0" (
            with pkgs.gst_all_1;
            [
              gstreamer
              gst-plugins-base
              gst-plugins-good
              gst-plugins-bad
              gst-plugins-ugly
              gst-plugins-rs
              gst-libav
            ]
          );

          hardeningDisable = [ "fortify" ]; # Required for Delve
          # For delve to work, you need to add the following line to your `programs.zed-editor`:
          # package = pkgs.zed-editor.fhs;
          buildInputs =
            with pkgs;
            [
              appstream
              delve
              go_1_26
              gopls
              gtk4
              librsvg
              libsecret
              graphviz
              glib-networking
              gst_all_1.gstreamer
              gst_all_1.gst-plugins-base
              gst_all_1.gst-plugins-good
              gst_all_1.gst-plugins-bad
              gst_all_1.gst-plugins-ugly
              gst_all_1.gst-plugins-rs
              gst_all_1.gst-libav
              pkg-config # Needed for the first compile with CGO
              sass
              cacert
            ]
            ++ pkgs.lib.optionals pkgs.stdenv.isLinux [
              flatpak-builder
            ];
        } // pkgs.lib.optionalAttrs pkgs.stdenv.isDarwin {
          # Only set GIO_EXTRA_MODULES on Darwin where the system gio module
          # path is unavailable. On Linux, leaving this unset lets gio find
          # the system glib-networking modules required for HTTPS / TLS in
          # souphttpsrc; setting it (even to empty) breaks network streaming.
          GIO_EXTRA_MODULES = "${pkgs.glib-networking}/lib/gio/modules";
          # Override after setup hooks (which propagate librsvg's broken cache).
          # Prepend libraryPath/share so GTK's icon theme loader picks up the
          # Adwaita icons that ship inside the symlinkJoin on macOS.
          shellHook = ''
            export GDK_PIXBUF_MODULE_FILE=${darwinPixbufLoadersCache}
            export DYLD_FALLBACK_LIBRARY_PATH=${libraryPath}/lib
            export XDG_DATA_DIRS=${libraryPath}/share''${XDG_DATA_DIRS:+:$XDG_DATA_DIRS}
          '';
        });

        packages.scanline = (pkgs.buildGoModule.override { go = pkgs.go_1_26; }) (finalAttrs: {
          pname = "scanline";
          inherit version;
          src = pkgs.lib.cleanSource ./.;
          vendorHash = "sha256-RQn9pK/jfkzvpJTG9xADz91W40Ss19JEtZf+0N+zLUA=";

          ldflags = [
            "-X \"github.com/0skillallluck/scanline/app/dialogs/about.Commit=${
              (if (self ? rev) then self.rev else "")
            }\""
            "-X \"github.com/0skillallluck/scanline/app/dialogs/about.Version=${finalAttrs.version}\""
          ];

          buildInputs = with pkgs; [
            glib-networking # TLS support for libsoup (HTTPS streaming)
            gst_all_1.gstreamer
            gst_all_1.gst-plugins-base
            gst_all_1.gst-plugins-good
            gst_all_1.gst-plugins-bad
            gst_all_1.gst-plugins-ugly
            gst_all_1.gst-plugins-rs
            gst_all_1.gst-libav
            libsecret
          ];
          doCheck = false;
          nativeBuildInputs = with pkgs; [
            pkg-config
            gtk4
            copyDesktopItems
            makeWrapper
            wrapGAppsHook4
          ];

          desktopItems = [
            (pkgs.makeDesktopItem {
              name = "dev.skillless.Scanline";
              exec = "scanline %u";
              icon = "dev.skillless.Scanline";
              comment = "An unofficial native GTK4 / Adwaita client for Plex";
              desktopName = "Scanline";
              mimeTypes = [
                "x-scheme-handler/plex"
              ];
              categories = [
                "AudioVideo"
                "Video"
                "GNOME"
                "GTK"
              ];
            })
          ];

          postInstall = ''
            wrapProgram $out/bin/scanline \
              --prefix GST_PLUGIN_PATH : "$GST_PLUGIN_SYSTEM_PATH_1_0" \
              --set-default PUREGOTK_LIB_FOLDER ${libraryPath}/lib \
              ''${gappsWrapperArgs[@]}
            install -Dm644 assets/icons/app.svg $out/share/icons/hicolor/scalable/apps/dev.skillless.Scanline.svg
            install -Dm644 assets/icons/app.png $out/share/icons/hicolor/128x128/apps/dev.skillless.Scanline.png
            install -Dm644 assets/icons/app-symbolic.svg $out/share/icons/hicolor/symbolic/apps/dev.skillless.Scanline-symbolic.svg
            install -Dm644 assets/meta/dev.skillless.Scanline.gschema.xml $out/share/glib-2.0/schemas/dev.skillless.Scanline.gschema.xml
            glib-compile-schemas $out/share/glib-2.0/schemas
          '';

          meta = {
            description = "Scanline is an unofficial native GTK4 / Adwaita client for Plex";
            homepage = "https://github.com/0skillallluck/scanline";
            license = pkgs.lib.licenses.gpl3Plus;
            maintainers = with pkgs.lib.maintainers; [
              drafolin
              nilathedragon
            ];
            mainProgram = "scanline";
          };
        });

        packages.default = self.packages.${system}.scanline;
      }
      (pkgs.lib.optionalAttrs pkgs.stdenv.isDarwin {
        packages.scanline-darwin-app = scanlineDarwinApp;
      })
    );
}
