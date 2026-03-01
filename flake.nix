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
          ];
        };
      in
      {
        devShell = pkgs.mkShell {
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
              gst_all_1.gst-libav
              pkg-config # Needed for the first compile with CGO
              sass
            ]
            ++ pkgs.lib.optionals pkgs.stdenv.isLinux [
              flatpak-builder
            ];
        };

        packages.scanline = (pkgs.buildGoModule.override { go = pkgs.go_1_26; }) (finalAttrs: {
          pname = "scanline";
          version = "0.1.0";
          src = pkgs.lib.cleanSource ./.;
          vendorHash = "sha256-GksJcnLcmHZKwhyHauHbGweZAOmQ3K/FV4FeMszpMYI=";

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
