#include <stdbool.h>
#import <Cocoa/Cocoa.h>
#import <dispatch/dispatch.h>

extern void darwinTrayReady(void);
extern void darwinTrayToggle(void);
extern void darwinTrayStopAll(void);
extern void darwinTrayQuit(void);

@interface ATStarterTrayController : NSObject
@property(strong) NSStatusItem *statusItem;
@property(strong) NSMenuItem *runningItem;
@property(strong) NSMenuItem *toggleItem;
@end

@implementation ATStarterTrayController

- (NSImage *)dockIconImageFromImage:(NSImage *)sourceImage {
	NSImage *dockImage = [[NSImage alloc] initWithSize:NSMakeSize(1024, 1024)];
	[dockImage lockFocus];

	[[NSColor clearColor] setFill];
	NSRectFill(NSMakeRect(0, 0, 1024, 1024));

	CGFloat inset = 82.0;
	NSRect drawRect = NSMakeRect(inset, inset, 1024 - inset * 2, 1024 - inset * 2);
	[sourceImage drawInRect:drawRect fromRect:NSZeroRect operation:NSCompositingOperationSourceOver fraction:1.0 respectFlipped:NO hints:@{NSImageHintInterpolation: @(NSImageInterpolationHigh)}];

	[dockImage unlockFocus];
	return dockImage;
}

- (void)setupWithIconData:(NSData *)iconData {
	self.statusItem = [[NSStatusBar systemStatusBar] statusItemWithLength:NSSquareStatusItemLength];
	self.statusItem.button.toolTip = @"AT Starter";
	if (iconData != nil) {
		NSImage *appIcon = [[NSImage alloc] initWithData:iconData];
		if (appIcon != nil) {
			[NSApp setApplicationIconImage:[self dockIconImageFromImage:appIcon]];
		}

		NSImage *image = [[NSImage alloc] initWithData:iconData];
		[image setSize:NSMakeSize(18, 18)];
		image.template = NO;
		self.statusItem.button.image = image;
		self.statusItem.button.imagePosition = NSImageOnly;
	} else {
		self.statusItem.button.title = @"AT";
	}

	NSMenu *menu = [[NSMenu alloc] initWithTitle:@"AT Starter"];
	menu.autoenablesItems = NO;

	self.runningItem = [[NSMenuItem alloc] initWithTitle:@"运行中: 0 个" action:nil keyEquivalent:@""];
	self.runningItem.enabled = NO;
	[menu addItem:self.runningItem];
	[menu addItem:[NSMenuItem separatorItem]];

	self.toggleItem = [[NSMenuItem alloc] initWithTitle:@"隐藏窗口" action:@selector(handleToggle:) keyEquivalent:@""];
	self.toggleItem.target = self;
	[menu addItem:self.toggleItem];

	NSMenuItem *stopAllItem = [[NSMenuItem alloc] initWithTitle:@"停止全部项目" action:@selector(handleStopAll:) keyEquivalent:@""];
	stopAllItem.target = self;
	[menu addItem:stopAllItem];

	[menu addItem:[NSMenuItem separatorItem]];

	NSMenuItem *quitItem = [[NSMenuItem alloc] initWithTitle:@"退出" action:@selector(handleQuit:) keyEquivalent:@""];
	quitItem.target = self;
	[menu addItem:quitItem];

	self.statusItem.menu = menu;
}

- (void)handleToggle:(id)sender {
	darwinTrayToggle();
}

- (void)handleStopAll:(id)sender {
	darwinTrayStopAll();
}

- (void)handleQuit:(id)sender {
	darwinTrayQuit();
}

- (void)updateRunning:(int)n {
	self.runningItem.title = [NSString stringWithFormat:@"运行中: %d 个", n];
	self.statusItem.button.toolTip = [NSString stringWithFormat:@"AT Starter - %d 个运行中", n];
}

- (void)updateVisible:(BOOL)visible {
	self.toggleItem.title = visible ? @"隐藏窗口" : @"显示窗口";
}

@end

static ATStarterTrayController *trayController = nil;

void atstarter_start_tray(const char *iconBytes, int length) {
	NSData *iconData = nil;
	if (iconBytes != NULL && length > 0) {
		iconData = [NSData dataWithBytes:iconBytes length:(NSUInteger)length];
	}

	dispatch_async(dispatch_get_main_queue(), ^{
		if (trayController == nil) {
			trayController = [[ATStarterTrayController alloc] init];
			[trayController setupWithIconData:iconData];
		}
		darwinTrayReady();
	});
}

void atstarter_update_running(int n) {
	dispatch_async(dispatch_get_main_queue(), ^{
		if (trayController != nil) {
			[trayController updateRunning:n];
		}
	});
}

void atstarter_update_visible(bool visible) {
	dispatch_async(dispatch_get_main_queue(), ^{
		if (trayController != nil) {
			[trayController updateVisible:visible];
		}
	});
}
