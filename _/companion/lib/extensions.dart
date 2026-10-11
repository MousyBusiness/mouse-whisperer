import 'dart:math' as math;
import 'dart:ui';

import 'package:flutter/material.dart';
import 'package:flutter_animate/flutter_animate.dart';
import 'package:mouse_companion/theme/app_colors.dart';

extension XWidget on Widget {


  Widget shimmer(Duration delay, Duration duration) {
    return animate(onComplete: (a) => a.repeat()).shimmer(
      colors: [
        Colors.black,
        Colors.white,
        Colors.black,
      ],
      curve: Curves.easeInOut,
      delay: delay,
      duration: duration,
    );
  }

  Widget onTapDown(VoidCallback callback, {HitTestBehavior? behavior}) {
    return GestureDetector(behavior: behavior, onTapDown: (_) => callback(), child: this);
  }

  Widget onSecondaryTapDown(VoidCallback callback, {HitTestBehavior? behavior}) {
    return GestureDetector(behavior: behavior, onSecondaryTapDown: (_) => callback(), child: this);
  }

  Widget onTap(VoidCallback callback, {HitTestBehavior? behavior}) {
    return GestureDetector(behavior: behavior, onTap: callback, child: this);
  }

  Widget onDoubleTap(VoidCallback callback, {HitTestBehavior? behavior}) {
    return GestureDetector(behavior: behavior, onDoubleTap: callback, child: this);
  }

  Widget onLongPress(VoidCallback callback) {
    return GestureDetector(onLongPress: callback, child: this);
  }

  Widget onLongPressStart(void Function(LongPressStartDetails details) callback) {
    return GestureDetector(onLongPressStart: (d) => callback(d), child: this);
  }

  Widget minHeight(double h) {
    return ConstrainedBox(
      constraints: BoxConstraints(minHeight: h),
      child: this,
    );
  }

  Widget minWidth(double w) {
    return ConstrainedBox(
      constraints: BoxConstraints(minWidth: w),
      child: this,
    );
  }

  Widget maxWidth(double w) {
    return ConstrainedBox(
      constraints: BoxConstraints(maxWidth: w),
      child: this,
    );
  }

  Widget maxHeight(double h) {
    return ConstrainedBox(
      constraints: BoxConstraints(maxHeight: h),
      child: this,
    );
  }

  Widget get noScrollbar {
    return ScrollConfiguration(behavior: GrabbableScrollBehavior().copyWith(scrollbars: false), child: this);
  }

  Widget get noScrollbarScrollable {
    return scrollable.noScrollbar;
  }

  /*
  CustomScrollView(physics: AlwaysScrollableScrollPhysics(), slivers: [
            SliverFillRemaining(
              fillOverscroll: true,
              child: IntrinsicHeight(
   */

  Widget get minSizeScrollable {
    return LayoutBuilder(
      builder: (context, constraints) {
        return ConstrainedBox(
          constraints: BoxConstraints(minHeight: constraints.maxHeight),
          child: noScrollbarScrollable,
        );
      },
    );
  }

  Widget get scrollable {
    return SingleChildScrollView(
      physics: const ClampingScrollPhysics(),
      // padding: const EdgeInsets.only(bottom: 200),
      child: this,
    );
  }

  Widget get scrollableNoClip {
    return SingleChildScrollView(
      clipBehavior: Clip.none,
      // padding: const EdgeInsets.only(bottom: 200),
      child: this,
    );
  }

  Widget get alignTopLeft {
    return Align(alignment: Alignment.topLeft, child: this);
  }

  Widget get alignTopRight {
    return Align(alignment: Alignment.topRight, child: this);
  }

  Widget get alignTop {
    return Align(alignment: Alignment.topCenter, child: this);
  }

  Widget get alignLeft {
    return Align(alignment: Alignment.centerLeft, child: this);
  }

  Widget get alignRight {
    return Align(alignment: Alignment.centerRight, child: this);
  }

  Widget get alignBottom {
    return Align(alignment: Alignment.bottomCenter, child: this);
  }

  Widget get alwaysScrollable {
    return SingleChildScrollView(
      physics: const AlwaysScrollableScrollPhysics(),
      padding: const EdgeInsets.only(bottom: 80),
      child: this,
    );
  }

  Widget refresh(Future Function() onRefresh) {
    return RefreshIndicator(onRefresh: onRefresh, child: this);
  }

  Widget refreshMinSize(Future Function() onRefresh) {
    return LayoutBuilder(
      builder: (context, constraints) {
        return ConstrainedBox(
          constraints: BoxConstraints(minHeight: constraints.maxHeight),
          child: alwaysScrollable.refresh(onRefresh),
        );
      },
    );
  }

  Widget refreshNonScrollingPage(Future Function() onRefresh) {
    return LayoutBuilder(
      builder: (context, constraints) =>
          SizedBox(height: constraints.maxHeight, child: this).scrollable.refresh(onRefresh),
    );
  }

  Widget get hscrollable {
    return SingleChildScrollView(scrollDirection: Axis.horizontal, child: this);
  }

  Widget maybeSafeArea(bool safe) {
    if (!safe) return this;
    return SafeArea(child: this);
  }

  Widget maybeExpand(bool expand) {
    if (!expand) return this;
    return Expanded(child: this);
  }

  Widget get aspect {
    return AspectRatio(aspectRatio: 1, child: this);
  }

  Widget get safeArea {
    return SafeArea(child: this);
  }

  Widget get center {
    return Center(child: this);
  }

  Widget get flex {
    return Flexible(child: this);
  }

  Widget get expand {
    return Expanded(child: this);
  }

  Widget ssz(double size) {
    return SizedBox(width: size, height: size, child: this);
  }

  Widget size(double? width, double? height) {
    return SizedBox(width: width, height: height, child: this);
  }

  Widget h(double v) {
    return SizedBox(height: v, child: this);
  }

  Widget w(double v) {
    return SizedBox(width: v, child: this);
  }

  Widget t(double v) {
    return Padding(
      padding: EdgeInsets.only(top: v),
      child: this,
    );
  }

  Widget l(double v) {
    return Padding(
      padding: EdgeInsets.only(left: v),
      child: this,
    );
  }

  Widget r(double v) {
    return Padding(
      padding: EdgeInsets.only(right: v),
      child: this,
    );
  }

  Widget b(double v) {
    return Padding(
      padding: EdgeInsets.only(bottom: v),
      child: this,
    );
  }

  Widget p(double v) {
    return Padding(padding: EdgeInsets.all(v), child: this);
  }

  Widget sliverp({double left = 0, double top = 0, double right = 0, double bottom = 0}) {
    return SliverPadding(
      padding: EdgeInsets.only(left: left, top: top, right: right, bottom: bottom),
      sliver: this,
    );
  }

  Widget get intrinsic {
    return IntrinsicHeight(child: IntrinsicWidth(child: this));
  }

  Widget get intrinsicH {
    return IntrinsicHeight(child: this);
  }

  Widget get intrinsicW {
    return IntrinsicWidth(child: this);
  }

  Widget pad({double left = 0, double top = 0, double right = 0, double bottom = 0}) {
    return Padding(
      padding: EdgeInsets.only(left: left, top: top, right: right, bottom: bottom),
      child: this,
    );
  }

  Widget pos({double? left, double? top, double? right, double? bottom}) {
    return Positioned(left: left, top: top, right: right, bottom: bottom, child: this);
  }

  Widget hero(String tag) {
    return Hero(tag: tag, child: this);
  }

  Widget posFill({double? left = 0, double? top = 0, double? right = 0, double? bottom = 0}) {
    return Positioned(left: left, top: top, right: right, bottom: bottom, child: this);
  }

  Widget get p4 {
    return Padding(padding: const EdgeInsets.all(4.0), child: this);
  }

  Widget get p6 {
    return Padding(padding: const EdgeInsets.all(6.0), child: this);
  }

  Widget get p8 {
    return Padding(padding: const EdgeInsets.all(8.0), child: this);
  }

  Widget get p16 {
    return Padding(padding: const EdgeInsets.all(16.0), child: this);
  }

  Widget get p24 {
    return Padding(padding: const EdgeInsets.all(24.0), child: this);
  }

  Widget get v4 {
    return Padding(padding: const EdgeInsets.symmetric(vertical: 4.0), child: this);
  }

  Widget get v6 {
    return Padding(padding: const EdgeInsets.symmetric(vertical: 6.0), child: this);
  }

  Widget get v8 {
    return Padding(padding: const EdgeInsets.symmetric(vertical: 8.0), child: this);
  }

  Widget get v16 {
    return Padding(padding: const EdgeInsets.symmetric(vertical: 16.0), child: this);
  }

  Widget get v32 {
    return Padding(padding: const EdgeInsets.symmetric(vertical: 32.0), child: this);
  }

  Widget get h2 {
    return Padding(padding: const EdgeInsets.symmetric(horizontal: 2.0), child: this);
  }

  Widget get h4 {
    return Padding(padding: const EdgeInsets.symmetric(horizontal: 4.0), child: this);
  }

  Widget get h8 {
    return Padding(padding: const EdgeInsets.symmetric(horizontal: 8.0), child: this);
  }

  Widget get h12 {
    return Padding(padding: const EdgeInsets.symmetric(horizontal: 12.0), child: this);
  }

  Widget get h16 {
    return Padding(padding: const EdgeInsets.symmetric(horizontal: 16.0), child: this);
  }

  Widget get h24 {
    return Padding(padding: const EdgeInsets.symmetric(horizontal: 24.0), child: this);
  }

  Widget get h32 {
    return Padding(padding: const EdgeInsets.symmetric(horizontal: 32.0), child: this);
  }

  Widget get h64 {
    return Padding(padding: const EdgeInsets.symmetric(horizontal: 64.0), child: this);
  }

  Widget get ignore {
    return IgnorePointer(child: this);
  }

  Widget get absorb {
    return AbsorbPointer(child: this);
  }

  Widget get noPopScaffold {
    return PopScope(canPop: false, child: Scaffold(body: this));
  }

  Widget get safeTransparentScaffold {
    return Scaffold(
      backgroundColor: Colors.transparent,
      body: SafeArea(child: this),
    );
  }

  Widget get fill {
    return Positioned.fill(child: this);
  }

  /*
CustomScrollView(
  slivers: <Widget>[
    SliverFillRemaining(
      hasScrollBody: false, // Set to true if this sliver itself should be scrollable
      child: Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: <Widget>[
            // Your widgets here
            Text('Centered Content in Custom Scroll View'),
          ],
        ),
      ),
    ),
  ],
)*/
  Widget get topStretch {
    return Positioned(top: 0, left: 0, right: 0, child: this);
  }

  Widget get bottomStretch {
    return Positioned(bottom: 0, left: 0, right: 0, child: this);
  }

  Widget get clipCard {
    return ClipRRect(borderRadius: BorderRadius.circular(8), child: this);
  }

  Widget colored(Color color) {
    return ColoredBox(color: color, child: this);
  }

  Widget get clipOval {
    return ClipOval(child: this);
  }

  Widget get card {
    return Container(
      decoration: ShapeDecoration(
        color: AppColors.codeBackground,
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
      ),
      child: this,
    );
  }

  Widget get rounded {
    return Container(
      decoration: ShapeDecoration(
        color: AppColors.darkGrey,
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
      ),
      child: this,
    );
  }

  Widget get roundedOutline {
    return Container(
      decoration: ShapeDecoration(
        color: AppColors.darkGrey,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(16),
          side: const BorderSide(color: AppColors.midGrey),
        ),
      ),
      child: this,
    );
  }

  Widget circle({Color? color}) {
    return Container(
      decoration: ShapeDecoration(color: color, shape: const CircleBorder()),
      child: this,
    );
  }

  Widget get stadium1 {
    return Container(
      decoration: ShapeDecoration(color: Colors.black.withValues(alpha: 0.5), shape: const StadiumBorder()),
      child: this,
    );
  }

  Widget get blur => SizedBox(
    child: ClipRRect(
      borderRadius: BorderRadius.circular(16),
      child: BackdropFilter(filter: ImageFilter.blur(sigmaX: 2, sigmaY: 2), child: this),
    ),
  );

  Widget get blurBackground => SizedBox(
    child: BackdropFilter(filter: ImageFilter.blur(sigmaX: 10, sigmaY: 10), child: this),
  );

  double _degreesToRads(double degrees) {
    return (math.pi / 180) * degrees;
  }

  Widget rotated(double degrees) {
    return Transform.rotate(angle: _degreesToRads(degrees), child: this);
  }

  Widget flip(bool flip) {
    if (!flip) return this;
    return Transform.scale(scaleX: -1, alignment: Alignment.center, child: this);
  }
}

class GrabbableScrollBehavior extends MaterialScrollBehavior {
  // Override behavior methods and getters like dragDevices
  @override
  Set<PointerDeviceKind> get dragDevices => {
    PointerDeviceKind.touch,
    PointerDeviceKind.mouse,
    PointerDeviceKind.trackpad,
    // etc.
  };
}
