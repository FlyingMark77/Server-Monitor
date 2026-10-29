#import <Cocoa/Cocoa.h>
extern void goStatusItemClicked(void);
extern void goSystemWoke(void);
@interface SMMenuHandler : NSObject
- (void)handleClick:(id)sender;
- (void)didWake:(NSNotification *)notification;
@end
@implementation SMMenuHandler
- (void)handleClick:(id)sender { goStatusItemClicked(); }
- (void)didWake:(NSNotification *)notification { goSystemWoke(); }
@end
static NSStatusItem *statusItem;
static SMMenuHandler *handler;
static NSString *pendingTitle;
int smAppHidden(void) {
 __block int hidden=0;
 dispatch_sync(dispatch_get_main_queue(), ^{ hidden=NSApp.isHidden; });
 return hidden;
}
void smSetupTray(void) {
 dispatch_async(dispatch_get_main_queue(), ^{
  handler = [[SMMenuHandler alloc] init];
  statusItem = [[[NSStatusBar systemStatusBar] statusItemWithLength:NSVariableStatusItemLength] retain];
  statusItem.button.title = pendingTitle ?: @"未配置";
  statusItem.button.toolTip = @"Server Monitor — 多服务器 GPU 监控";
  statusItem.button.action = @selector(handleClick:);
  statusItem.button.target = handler;
  [[NSWorkspace sharedWorkspace].notificationCenter addObserver:handler selector:@selector(didWake:) name:NSWorkspaceDidWakeNotification object:nil];
 });
}
void smSetTitle(const char *title) {
 NSString *copy = [[NSString alloc] initWithUTF8String:title];
 dispatch_async(dispatch_get_main_queue(), ^{
  [pendingTitle release]; pendingTitle = copy;
  statusItem.button.title = pendingTitle;
 });
}
void smPopupPosition(int width, int *x, int *y) {
 __block int px=20, py=32;
 dispatch_sync(dispatch_get_main_queue(), ^{
  if (!statusItem.button.window) return;
  NSRect rect = [statusItem.button convertRect:statusItem.button.bounds toView:nil];
  rect = [statusItem.button.window convertRectToScreen:rect];
  NSScreen *screen = statusItem.button.window.screen ?: NSScreen.mainScreen;
  NSRect area = screen.visibleFrame;
  px = (int)MAX(NSMinX(area), MIN(NSMaxX(rect)-width, NSMaxX(area)-width));
  CGFloat primaryHeight = NSMaxY([NSScreen.screens.firstObject frame]);
  py = (int)(primaryHeight-NSMinY(rect)+6);
 });
 *x=px;*y=py;
}
