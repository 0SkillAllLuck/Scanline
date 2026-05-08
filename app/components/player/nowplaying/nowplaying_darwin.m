//go:build darwin

#import <Foundation/Foundation.h>
#import <AppKit/AppKit.h>
#import <MediaPlayer/MediaPlayer.h>
#include "_cgo_export.h"

// Command IDs — must stay in sync with cmd* iota in nowplaying_darwin.go.
#define CMD_PLAY_PAUSE 0
#define CMD_PLAY       1
#define CMD_PAUSE      2
#define CMD_NEXT       3
#define CMD_PREV       4
#define CMD_SKIP_FWD   5
#define CMD_SKIP_BACK  6
#define CMD_SEEK_TO    7
#define CMD_STOP       8

// sInfo accumulates the dictionary we publish to nowPlayingInfo. We mutate
// it in place across metadata / state / position / artwork updates, then
// re-assign nowPlayingInfo to push the new snapshot to the framework.
static NSMutableDictionary<NSString *, id> *sInfo = nil;

static MPNowPlayingInfoCenter *infoCenter(void) {
    return MPNowPlayingInfoCenter.defaultCenter;
}

void scanline_np_init(void) {
    sInfo = [NSMutableDictionary dictionary];
    MPRemoteCommandCenter *cc = MPRemoteCommandCenter.sharedCommandCenter;

    // Each block captures only the integer command ID (a primitive value
    // type). The block is retained by the MPRemoteCommand and invoked on
    // the AppKit main runloop; scanlineNpDispatch hops to the GLib main
    // loop before invoking Go-side handlers. Targets are installed once
    // for the lifetime of the process; per-session state lives Go-side
    // behind the `current` pointer in nowplaying_darwin.go.
    [cc.togglePlayPauseCommand addTargetWithHandler:^MPRemoteCommandHandlerStatus(MPRemoteCommandEvent *event) {
        scanlineNpDispatch(CMD_PLAY_PAUSE, 0.0);
        return MPRemoteCommandHandlerStatusSuccess;
    }];
    [cc.playCommand addTargetWithHandler:^MPRemoteCommandHandlerStatus(MPRemoteCommandEvent *event) {
        scanlineNpDispatch(CMD_PLAY, 0.0);
        return MPRemoteCommandHandlerStatusSuccess;
    }];
    [cc.pauseCommand addTargetWithHandler:^MPRemoteCommandHandlerStatus(MPRemoteCommandEvent *event) {
        scanlineNpDispatch(CMD_PAUSE, 0.0);
        return MPRemoteCommandHandlerStatusSuccess;
    }];
    [cc.nextTrackCommand addTargetWithHandler:^MPRemoteCommandHandlerStatus(MPRemoteCommandEvent *event) {
        scanlineNpDispatch(CMD_NEXT, 0.0);
        return MPRemoteCommandHandlerStatusSuccess;
    }];
    [cc.previousTrackCommand addTargetWithHandler:^MPRemoteCommandHandlerStatus(MPRemoteCommandEvent *event) {
        scanlineNpDispatch(CMD_PREV, 0.0);
        return MPRemoteCommandHandlerStatusSuccess;
    }];

    cc.skipForwardCommand.preferredIntervals = @[@15];
    [cc.skipForwardCommand addTargetWithHandler:^MPRemoteCommandHandlerStatus(MPRemoteCommandEvent *event) {
        double interval = 15.0;
        if ([event isKindOfClass:[MPSkipIntervalCommandEvent class]]) {
            interval = ((MPSkipIntervalCommandEvent *)event).interval;
        }
        scanlineNpDispatch(CMD_SKIP_FWD, interval);
        return MPRemoteCommandHandlerStatusSuccess;
    }];
    cc.skipBackwardCommand.preferredIntervals = @[@15];
    [cc.skipBackwardCommand addTargetWithHandler:^MPRemoteCommandHandlerStatus(MPRemoteCommandEvent *event) {
        double interval = 15.0;
        if ([event isKindOfClass:[MPSkipIntervalCommandEvent class]]) {
            interval = ((MPSkipIntervalCommandEvent *)event).interval;
        }
        scanlineNpDispatch(CMD_SKIP_BACK, interval);
        return MPRemoteCommandHandlerStatusSuccess;
    }];

    [cc.changePlaybackPositionCommand addTargetWithHandler:^MPRemoteCommandHandlerStatus(MPRemoteCommandEvent *event) {
        double positionTime = 0.0;
        if ([event isKindOfClass:[MPChangePlaybackPositionCommandEvent class]]) {
            positionTime = ((MPChangePlaybackPositionCommandEvent *)event).positionTime;
        }
        scanlineNpDispatch(CMD_SEEK_TO, positionTime);
        return MPRemoteCommandHandlerStatusSuccess;
    }];

    [cc.stopCommand addTargetWithHandler:^MPRemoteCommandHandlerStatus(MPRemoteCommandEvent *event) {
        scanlineNpDispatch(CMD_STOP, 0.0);
        return MPRemoteCommandHandlerStatusSuccess;
    }];
}

void scanline_np_set_metadata(const char *title, const char *artist,
    const char *album, double durSec, int kind) {
    @autoreleasepool {
        if (title && *title) {
            sInfo[MPMediaItemPropertyTitle] = [NSString stringWithUTF8String:title];
        }
        if (artist && *artist) {
            sInfo[MPMediaItemPropertyArtist] = [NSString stringWithUTF8String:artist];
        } else {
            [sInfo removeObjectForKey:MPMediaItemPropertyArtist];
        }
        if (album && *album) {
            sInfo[MPMediaItemPropertyAlbumTitle] = [NSString stringWithUTF8String:album];
        } else {
            [sInfo removeObjectForKey:MPMediaItemPropertyAlbumTitle];
        }
        if (durSec > 0) {
            sInfo[MPMediaItemPropertyPlaybackDuration] = @(durSec);
        }
        sInfo[MPNowPlayingInfoPropertyMediaType] = @(MPNowPlayingInfoMediaTypeVideo);
        infoCenter().nowPlayingInfo = sInfo;
    }
}

void scanline_np_set_state(int state) {
    MPNowPlayingPlaybackState s = MPNowPlayingPlaybackStateUnknown;
    double rate = 0.0;
    switch (state) {
        case 0: s = MPNowPlayingPlaybackStatePlaying; rate = 1.0; break;
        case 1: s = MPNowPlayingPlaybackStatePaused;  rate = 0.0; break;
        case 2: s = MPNowPlayingPlaybackStateStopped; rate = 0.0; break;
    }
    infoCenter().playbackState = s;
    if (sInfo) {
        sInfo[MPNowPlayingInfoPropertyPlaybackRate] = @(rate);
        infoCenter().nowPlayingInfo = sInfo;
    }
}

void scanline_np_set_position(double posSec, double rate) {
    if (sInfo) {
        sInfo[MPNowPlayingInfoPropertyElapsedPlaybackTime] = @(posSec);
        sInfo[MPNowPlayingInfoPropertyPlaybackRate] = @(rate);
        infoCenter().nowPlayingInfo = sInfo;
    }
}

void scanline_np_set_artwork(const void *bytes, int len) {
    @autoreleasepool {
        NSData *data = [NSData dataWithBytes:bytes length:len];
        NSImage *img = [[NSImage alloc] initWithData:data];
        if (!img) return;
        // ARC keeps `img` alive via the requestHandler block's capture;
        // MPMediaItemArtwork retains the block; sInfo retains the artwork;
        // setting nowPlayingInfo = nil in scanline_np_clear breaks the
        // chain and releases everything.
        MPMediaItemArtwork *art = [[MPMediaItemArtwork alloc]
            initWithBoundsSize:img.size
            requestHandler:^NSImage *(CGSize size) {
                return img;
            }];
        if (sInfo) {
            sInfo[MPMediaItemPropertyArtwork] = art;
            infoCenter().nowPlayingInfo = sInfo;
        }
    }
}

void scanline_np_set_handler_enabled(int handlerID, bool enabled) {
    MPRemoteCommandCenter *cc = MPRemoteCommandCenter.sharedCommandCenter;
    MPRemoteCommand *cmd = nil;
    switch (handlerID) {
        case CMD_PLAY_PAUSE: cmd = cc.togglePlayPauseCommand; break;
        case CMD_PLAY:       cmd = cc.playCommand; break;
        case CMD_PAUSE:      cmd = cc.pauseCommand; break;
        case CMD_NEXT:       cmd = cc.nextTrackCommand; break;
        case CMD_PREV:       cmd = cc.previousTrackCommand; break;
        case CMD_SKIP_FWD:   cmd = cc.skipForwardCommand; break;
        case CMD_SKIP_BACK:  cmd = cc.skipBackwardCommand; break;
        case CMD_SEEK_TO:    cmd = cc.changePlaybackPositionCommand; break;
        case CMD_STOP:       cmd = cc.stopCommand; break;
    }
    if (cmd) cmd.enabled = enabled;
}

void scanline_np_clear(void) {
    infoCenter().nowPlayingInfo = nil;
    infoCenter().playbackState = MPNowPlayingPlaybackStateStopped;
    [sInfo removeAllObjects];
    // Disable every command — targets stay installed so the next Configure
    // can re-enable cheaply without re-registering blocks.
    MPRemoteCommandCenter *cc = MPRemoteCommandCenter.sharedCommandCenter;
    cc.playCommand.enabled = NO;
    cc.pauseCommand.enabled = NO;
    cc.togglePlayPauseCommand.enabled = NO;
    cc.nextTrackCommand.enabled = NO;
    cc.previousTrackCommand.enabled = NO;
    cc.skipForwardCommand.enabled = NO;
    cc.skipBackwardCommand.enabled = NO;
    cc.changePlaybackPositionCommand.enabled = NO;
    cc.stopCommand.enabled = NO;
}
