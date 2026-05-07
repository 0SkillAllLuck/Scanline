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
        pkgs = import nixpkgs { inherit system; };
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
      in
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
          version = "0.4.0";
          src = pkgs.lib.cleanSource ./.;
          vendorHash = "sha256-zp+DQoo5jwJJrvC6KGdWBAvWtR55+6jslVlAAkfcU1U=";

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
    );
}
