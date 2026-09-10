export const isMobile = useMediaQuery("(max-width: 770px)");

/** True where a pointer can actually rest on something. Touch screens report
 *  `none`, and there a hover menu opens on tap-down and closes again on
 *  tap-up, which reads as a button that does nothing. */
export const canHover = useMediaQuery("(hover: hover)");
