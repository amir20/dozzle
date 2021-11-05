<script>
import { h } from "vue";
export default {
  name: "splitpanes",
  props: {
    horizontal: { type: Boolean },
    pushOtherPanes: { type: Boolean, default: true },
    dblClickSplitter: { type: Boolean, default: true },
    rtl: { type: Boolean, default: false }, // Right to left direction.
    firstSplitter: { type: Boolean },
  },

  provide() {
    return {
      requestUpdate: this.requestUpdate,
      onPaneAdd: this.onPaneAdd,
      onPaneRemove: this.onPaneRemove,
      onPaneClick: this.onPaneClick,
    };
  },

  data: () => ({
    container: null,
    ready: false,
    panes: [],
    touch: {
      mouseDown: false,
      dragging: false,
      activeSplitter: null,
    },
    splitterTaps: {
      // Used to detect double click on touch devices.
      splitter: null,
      timeoutId: null,
    },
  }),

  computed: {
    panesCount() {
      return this.panes.length;
    },
    // Indexed panes by `_uid` of Pane components for fast lookup.
    // Every time a pane is destroyed this index is recomputed.
    indexedPanes() {
      return this.panes.reduce((obj, pane) => (obj[pane.id] = pane) && obj, {});
    },
  },

  methods: {
    updatePaneComponents() {
      // On update refresh the size of each pane through the registered `update` method (in onPaneAdd).
      this.panes.forEach((pane) => {
        pane.update &&
          pane.update({
            // Panes are indexed by Pane component uid, as they might be inserted at different index.
            [this.horizontal ? "height" : "width"]: `${this.indexedPanes[pane.id].size}%`,
          });
      });
    },

    bindEvents() {
      document.addEventListener("mousemove", this.onMouseMove, { passive: false });
      document.addEventListener("mouseup", this.onMouseUp);

      // Passive: false to prevent scrolling while touch dragging.
      if ("ontouchstart" in window) {
        document.addEventListener("touchmove", this.onMouseMove, { passive: false });
        document.addEventListener("touchend", this.onMouseUp);
      }
    },

    unbindEvents() {
      document.removeEventListener("mousemove", this.onMouseMove, { passive: false });
      document.removeEventListener("mouseup", this.onMouseUp);

      if ("ontouchstart" in window) {
        document.removeEventListener("touchmove", this.onMouseMove, { passive: false });
        document.removeEventListener("touchend", this.onMouseUp);
      }
    },

    onMouseDown(event, splitterIndex) {
      this.bindEvents();
      this.touch.mouseDown = true;
      this.touch.activeSplitter = splitterIndex;
    },

    onMouseMove(event) {
      if (this.touch.mouseDown) {
        // Prevent scrolling while touch dragging (only works with an active event, eg. passive: false).
        event.preventDefault();
        this.touch.dragging = true;
        this.calculatePanesSize(this.getCurrentMouseDrag(event));
        this.$emit(
          "resize",
          this.panes.map((pane) => ({ min: pane.min, max: pane.max, size: pane.size }))
        );
      }
    },

    onMouseUp() {
      if (this.touch.dragging) {
        this.$emit(
          "resized",
          this.panes.map((pane) => ({ min: pane.min, max: pane.max, size: pane.size }))
        );
      }
      this.touch.mouseDown = false;
      // Keep dragging flag until click event is finished (click happens immediately after mouseup)
      // in order to prevent emitting `splitter-click` event if splitter was dragged.
      setTimeout(() => {
        this.touch.dragging = false;
        this.unbindEvents();
      }, 100);
    },

    // If touch device, detect double tap manually (2 taps separated by less than 500ms).
    onSplitterClick(event, splitterIndex) {
      if ("ontouchstart" in window) {
        event.preventDefault();

        // Detect splitter double taps if the option is on.
        if (this.dblClickSplitter) {
          if (this.splitterTaps.splitter === splitterIndex) {
            clearTimeout(this.splitterTaps.timeoutId);
            this.splitterTaps.timeoutId = null;
            this.onSplitterDblClick(event, splitterIndex);
            this.splitterTaps.splitter = null; // Reset for the next tap check.
          } else {
            this.splitterTaps.splitter = splitterIndex;
            this.splitterTaps.timeoutId = setTimeout(() => {
              this.splitterTaps.splitter = null;
            }, 500);
          }
        }
      }

      if (!this.touch.dragging) this.$emit("splitter-click", this.panes[splitterIndex]);
    },

    // On splitter dbl click or dbl tap maximize this pane.
    onSplitterDblClick(event, splitterIndex) {
      let totalMinSizes = 0;
      this.panes = this.panes.map((pane, i) => {
        pane.size = i === splitterIndex ? pane.max : pane.min;
        if (i !== splitterIndex) totalMinSizes += pane.min;

        return pane;
      });
      this.panes[splitterIndex].size -= totalMinSizes;
      this.$emit("pane-maximize", this.panes[splitterIndex]);
    },

    onPaneClick(event, paneId) {
      this.$emit("pane-click", this.indexedPanes[paneId]);
    },

    // Get the cursor position relative to the splitpane container.
    getCurrentMouseDrag(event) {
      const rect = this.container.getBoundingClientRect();
      const { clientX, clientY } = "ontouchstart" in window && event.touches ? event.touches[0] : event;

      return {
        x: clientX - rect.left,
        y: clientY - rect.top,
      };
    },

    // Returns the drag percentage of the splitter relative to the 2 panes it's inbetween.
    // if the sum of size of the 2 cells is 60%, the dragPercentage range will be 0 to 100% of this 60%.
    getCurrentDragPercentage(drag) {
      drag = drag[this.horizontal ? "y" : "x"];
      // In the code bellow 'size' refers to 'width' for vertical and 'height' for horizontal layout.
      const containerSize = this.container[this.horizontal ? "clientHeight" : "clientWidth"];
      if (this.rtl && !this.horizontal) drag = containerSize - drag;

      return (drag * 100) / containerSize;
    },

    calculatePanesSize(drag) {
      const splitterIndex = this.touch.activeSplitter;
      let sums = {
        prevPanesSize: this.sumPrevPanesSize(splitterIndex),
        nextPanesSize: this.sumNextPanesSize(splitterIndex),
        prevReachedMinPanes: 0,
        nextReachedMinPanes: 0,
      };

      const minDrag = 0 + (this.pushOtherPanes ? 0 : sums.prevPanesSize);
      const maxDrag = 100 - (this.pushOtherPanes ? 0 : sums.nextPanesSize);
      const dragPercentage = Math.max(Math.min(this.getCurrentDragPercentage(drag), maxDrag), minDrag);

      // If not pushing other panes, panes to resize are right before and right after splitter.
      let panesToResize = [splitterIndex, splitterIndex + 1];
      let paneBefore = this.panes[panesToResize[0]] || null;
      let paneAfter = this.panes[panesToResize[1]] || null;

      const paneBeforeMaxReached = paneBefore.max < 100 && dragPercentage >= paneBefore.max + sums.prevPanesSize;
      const paneAfterMaxReached =
        paneAfter.max < 100 && dragPercentage <= 100 - (paneAfter.max + this.sumNextPanesSize(splitterIndex + 1));
      // Prevent dragging beyond pane max.
      if (paneBeforeMaxReached || paneAfterMaxReached) {
        if (paneBeforeMaxReached) {
          paneBefore.size = paneBefore.max;
          paneAfter.size = Math.max(100 - paneBefore.max - sums.prevPanesSize - sums.nextPanesSize, 0);
        } else {
          paneBefore.size = Math.max(
            100 - paneAfter.max - sums.prevPanesSize - this.sumNextPanesSize(splitterIndex + 1),
            0
          );
          paneAfter.size = paneAfter.max;
        }
        return;
      }

      // When pushOtherPanes = true, find the closest expanded pane on each side of the splitter.
      if (this.pushOtherPanes) {
        const vars = this.doPushOtherPanes(sums, dragPercentage);
        if (!vars) return; // Prevent other calculation.

        ({ sums, panesToResize } = vars);
        paneBefore = this.panes[panesToResize[0]] || null;
        paneAfter = this.panes[panesToResize[1]] || null;
      }

      if (paneBefore !== null) {
        paneBefore.size = Math.min(
          Math.max(dragPercentage - sums.prevPanesSize - sums.prevReachedMinPanes, paneBefore.min),
          paneBefore.max
        );
      }
      if (paneAfter !== null) {
        paneAfter.size = Math.min(
          Math.max(100 - dragPercentage - sums.nextPanesSize - sums.nextReachedMinPanes, paneAfter.min),
          paneAfter.max
        );
      }
    },

    doPushOtherPanes(sums, dragPercentage) {
      const splitterIndex = this.touch.activeSplitter;
      const panesToResize = [splitterIndex, splitterIndex + 1];
      // Pushing Down.
      // Going smaller than the current pane min size: take the previous expanded pane.
      if (dragPercentage < sums.prevPanesSize + this.panes[panesToResize[0]].min) {
        panesToResize[0] = this.findPrevExpandedPane(splitterIndex).index;

        sums.prevReachedMinPanes = 0;
        // If pushing a n-2 or less pane, from splitter, then make sure all in between is at min size.
        if (panesToResize[0] < splitterIndex) {
          this.panes.forEach((pane, i) => {
            if (i > panesToResize[0] && i <= splitterIndex) {
              pane.size = pane.min;
              sums.prevReachedMinPanes += pane.min;
            }
          });
        }
        sums.prevPanesSize = this.sumPrevPanesSize(panesToResize[0]);
        // If nothing else to push down, cancel dragging.
        if (panesToResize[0] === undefined) {
          sums.prevReachedMinPanes = 0;
          this.panes[0].size = this.panes[0].min;
          this.panes.forEach((pane, i) => {
            if (i > 0 && i <= splitterIndex) {
              pane.size = pane.min;
              sums.prevReachedMinPanes += pane.min;
            }
          });
          this.panes[panesToResize[1]].size =
            100 - sums.prevReachedMinPanes - this.panes[0].min - sums.prevPanesSize - sums.nextPanesSize;
          return null;
        }
      }
      // Pushing Up.
      // Pushing up beyond min size is reached: take the next expanded pane.
      if (dragPercentage > 100 - sums.nextPanesSize - this.panes[panesToResize[1]].min) {
        panesToResize[1] = this.findNextExpandedPane(splitterIndex).index;
        sums.nextReachedMinPanes = 0;
        // If pushing a n+2 or more pane, from splitter, then make sure all in between is at min size.
        if (panesToResize[1] > splitterIndex + 1) {
          this.panes.forEach((pane, i) => {
            if (i > splitterIndex && i < panesToResize[1]) {
              pane.size = pane.min;
              sums.nextReachedMinPanes += pane.min;
            }
          });
        }
        sums.nextPanesSize = this.sumNextPanesSize(panesToResize[1] - 1);
        // If nothing else to push up, cancel dragging.
        if (panesToResize[1] === undefined) {
          sums.nextReachedMinPanes = 0;
          this.panes[this.panesCount - 1].size = this.panes[this.panesCount - 1].min;
          this.panes.forEach((pane, i) => {
            if (i < this.panesCount - 1 && i >= splitterIndex + 1) {
              pane.size = pane.min;
              sums.nextReachedMinPanes += pane.min;
            }
          });
          this.panes[panesToResize[0]].size =
            100 -
            sums.prevPanesSize -
            sums.nextReachedMinPanes -
            this.panes[this.panesCount - 1].min -
            sums.nextPanesSize;
          return null;
        }
      }
      return { sums, panesToResize };
    },

    sumPrevPanesSize(splitterIndex) {
      return this.panes.reduce((total, pane, i) => total + (i < splitterIndex ? pane.size : 0), 0);
    },

    sumNextPanesSize(splitterIndex) {
      return this.panes.reduce((total, pane, i) => total + (i > splitterIndex + 1 ? pane.size : 0), 0);
    },

    // Return the previous pane from siblings which has a size (width for vert or height for horz) of more than 0.
    findPrevExpandedPane(splitterIndex) {
      const pane = [...this.panes].reverse().find((p) => p.index < splitterIndex && p.size > p.min);
      return pane || {};
    },

    // Return the next pane from siblings which has a size (width for vert or height for horz) of more than 0.
    findNextExpandedPane(splitterIndex) {
      const pane = this.panes.find((p) => p.index > splitterIndex + 1 && p.size > p.min);
      return pane || {};
    },

    checkSplitpanesNodes() {
      const children = Array.from(this.container.children);
      children.forEach((child) => {
        const isPane = child.classList.contains("splitpanes__pane");
        const isSplitter = child.classList.contains("splitpanes__splitter");

        // Node is not a Pane or a splitter: remove it.
        if (!isPane && !isSplitter) {
          child.parentNode.removeChild(child); // el.remove() doesn't work on IE11.
          // eslint-disable-next-line no-console
          console.warn(
            "Splitpanes: Only <pane> elements are allowed at the root of <splitpanes>. One of your DOM nodes was removed."
          );
          return;
        }
      });
    },

    addSplitter(paneIndex, nextPaneNode, isVeryFirst = false) {
      const splitterIndex = paneIndex - 1;
      const elm = document.createElement("div");
      elm.classList.add("splitpanes__splitter");

      if (!isVeryFirst) {
        elm.onmousedown = (event) => this.onMouseDown(event, splitterIndex);

        if (typeof window !== "undefined" && "ontouchstart" in window) {
          elm.ontouchstart = (event) => this.onMouseDown(event, splitterIndex);
        }
        elm.onclick = (event) => this.onSplitterClick(event, splitterIndex + 1);
      }

      if (this.dblClickSplitter) {
        elm.ondblclick = (event) => this.onSplitterDblClick(event, splitterIndex + 1);
      }

      nextPaneNode.parentNode.insertBefore(elm, nextPaneNode);
    },

    removeSplitter(node) {
      node.onmousedown = undefined;
      node.onclick = undefined;
      node.ondblclick = undefined;
      node.parentNode.removeChild(node); // el.remove() doesn't work on IE11.
    },

    redoSplitters() {
      const children = Array.from(this.container.children);
      children.forEach((el) => {
        if (el.className.includes("splitpanes__splitter")) this.removeSplitter(el);
      });
      let paneIndex = 0;
      children.forEach((el) => {
        if (el.className.includes("splitpanes__pane")) {
          if (!paneIndex && this.firstSplitter) this.addSplitter(paneIndex, el, true);
          else if (paneIndex) this.addSplitter(paneIndex, el);
          paneIndex++;
        }
      });
    },

    // Called by Pane component on programmatic resize.
    requestUpdate({ target, ...args }) {
      const pane = this.indexedPanes[target._uid];
      Object.entries(args).forEach(([key, value]) => (pane[key] = value));
    },

    onPaneAdd(pane) {
      // 1. Add pane to array at the same index it was inserted in the <splitpanes> tag.
      let index = -1;
      Array.from(pane.$el.parentNode.children).some((el) => {
        if (el.className.includes("splitpanes__pane")) index++;
        return el === pane.$el;
      });

      const min = parseFloat(pane.minSize);
      const max = parseFloat(pane.maxSize);
      this.panes.splice(index, 0, {
        id: pane._uid,
        index,
        min: isNaN(min) ? 0 : min,
        max: isNaN(max) ? 100 : max,
        size: pane.size === null ? null : parseFloat(pane.size),
        givenSize: pane.size,
        update: pane.update,
      });

      // Redo indexes after insertion for other shifted panes.
      this.panes.forEach((p, i) => (p.index = i));

      if (this.ready) {
        this.$nextTick(() => {
          // 2. Add the splitter.
          this.redoSplitters();

          // 3. Resize the panes.
          this.resetPaneSizes({ addedPane: this.panes[index] });

          // 4. Fire `pane-add` event.
          this.$emit("pane-add", {
            index,
            panes: this.panes.map((pane) => ({ min: pane.min, max: pane.max, size: pane.size })),
          });
        });
      }
    },

    onPaneRemove(pane) {
      // 1. Remove the pane from array and redo indexes.
      const index = this.panes.findIndex((p) => p.id === pane._uid);
      const removed = this.panes.splice(index, 1)[0];
      this.panes.forEach((p, i) => (p.index = i));

      this.$nextTick(() => {
        // 2. Remove the splitter.
        this.redoSplitters();

        // 3. Resize the panes.
        this.resetPaneSizes({ removedPane: { ...removed, index } });

        // 4. Fire `pane-remove` event.
        this.$emit("pane-remove", {
          removed,
          panes: this.panes.map((pane) => ({ min: pane.min, max: pane.max, size: pane.size })),
        });
      });
    },

    resetPaneSizes(changedPanes = {}) {
      if (!changedPanes.addedPane && !changedPanes.removedPane) this.initialPanesSizing();
      else if (this.panes.some((pane) => pane.givenSize !== null || pane.min || pane.max < 100))
        this.equalizeAfterAddOrRemove(changedPanes);
      else this.equalize();

      if (this.ready)
        this.$emit(
          "resized",
          this.panes.map((pane) => ({ min: pane.min, max: pane.max, size: pane.size }))
        );
    },

    equalize() {
      const equalSpace = 100 / this.panesCount;
      let leftToAllocate = 0;
      let ungrowable = [];
      let unshrinkable = [];

      this.panes.forEach((pane) => {
        pane.size = Math.max(Math.min(equalSpace, pane.max), pane.min);

        leftToAllocate -= pane.size;
        if (pane.size >= pane.max) ungrowable.push(pane.id);
        if (pane.size <= pane.min) unshrinkable.push(pane.id);
      });

      if (leftToAllocate > 0.1) this.readjustSizes(leftToAllocate, ungrowable, unshrinkable);
    },

    initialPanesSizing() {
      let equalSpace = 100 / this.panesCount;
      let leftToAllocate = 100;
      let ungrowable = [];
      let unshrinkable = [];
      let definedSizes = 0;

      // Check if pre-allocated space is 100%.
      this.panes.forEach((pane) => {
        leftToAllocate -= pane.size;
        if (pane.size !== null) definedSizes++;
        if (pane.size >= pane.max) ungrowable.push(pane.id);
        if (pane.size <= pane.min) unshrinkable.push(pane.id);
      });

      // set pane sizes if not set.
      let leftToAllocate2 = 100;
      if (leftToAllocate > 0.1) {
        this.panes.forEach((pane) => {
          if (pane.size === null) {
            pane.size = Math.max(Math.min(leftToAllocate / (this.panesCount - definedSizes), pane.max), pane.min);
          }
          leftToAllocate2 -= pane.size;
        });

        if (leftToAllocate2 > 0.1) this.readjustSizes(leftToAllocate, ungrowable, unshrinkable);
      }
    },

    equalizeAfterAddOrRemove({ addedPane, removedPane } = {}) {
      let equalSpace = 100 / this.panesCount;
      let leftToAllocate = 0;
      let ungrowable = [];
      let unshrinkable = [];

      if (addedPane && addedPane.givenSize !== null) {
        equalSpace = (100 - addedPane.givenSize) / (this.panesCount - 1);
      }

      // Check if pre-allocated space is 100%.
      this.panes.forEach((pane) => {
        leftToAllocate -= pane.size;
        if (pane.size >= pane.max) ungrowable.push(pane.id);
        if (pane.size <= pane.min) unshrinkable.push(pane.id);
      });

      if (Math.abs(leftToAllocate) < 0.1) return; // Ok.

      this.panes.forEach((pane) => {
        if (addedPane && addedPane.givenSize !== null && addedPane.id === pane.id) {
        } else pane.size = Math.max(Math.min(equalSpace, pane.max), pane.min);

        leftToAllocate -= pane.size;
        if (pane.size >= pane.max) ungrowable.push(pane.id);
        if (pane.size <= pane.min) unshrinkable.push(pane.id);
      });

      if (leftToAllocate > 0.1) this.readjustSizes(leftToAllocate, ungrowable, unshrinkable);
    },

    /* recalculatePaneSizes ({ addedPane, removedPane } = {}) {
      let leftToAllocate = 100
      let equalSpaceToAllocate = leftToAllocate / this.panesCount
      let ungrowable = []
      let unshrinkable = []

      // When adding a pane with no size, apply min-size if defined otherwise divide another pane
      // (next or prev) in 2.
      // if (addedPane && addedPane.size === null) {
      //   if (addedPane.min) addedPane.size = addedPane.min
      //   else {
      //     const paneToDivide = this.panes[addedPane.index + 1] || this.panes[addedPane.index - 1]
      //     if (paneToDivide) {
      //       // @todo: Dividing that pane in 2 could be incorrect if becoming lower than its min size.
      //       addedPane.size = paneToDivide.size / 2
      //       paneToDivide.size /= 2
      //     }
      //   }
      // }

      this.panes.forEach((pane, i) => {
        // Added pane - reduce the size of the next pane.
        if (addedPane && addedPane.index + 1 === i) {
          pane.size = Math.max(Math.min(100 - this.sumPrevPanesSize(i) - this.sumNextPanesSize(i + 1), pane.max), pane.min)
          // @todo: if could not allocate correctly, try to allocate in the next pane straight away,
          // then still do the second loop if not correct.
        }

        // Removed pane - increase the size of the next pane.
        else if (removedPane && removedPane.index === i) {
          pane.size = Math.max(Math.min(100 - this.sumPrevPanesSize(i) - this.sumNextPanesSize(i + 1), pane.max), pane.min)
          // @todo: if could not allocate correctly, try to allocate in the next pane straight away,
          // then still do the second loop if not correct.
        }

        // Initial load and on demand recalculation.
        else if (!addedPane && !removedPane && pane.size === null) {
          pane.size = Math.max(Math.min(equalSpaceToAllocate, pane.max), pane.min)
        }

        leftToAllocate -= pane.size

        if (pane.size >= pane.max) ungrowable.push(pane.id)
        if (pane.size <= pane.min) unshrinkable.push(pane.id)
      })

      // Do one more loop to adjust sizes if still wrong.
      // > 0.1: Prevent maths rounding issues due to bytes.
      if (Math.abs(leftToAllocate) > 0.1) this.readjustSizes(leftToAllocate, ungrowable, unshrinkable)
    }, */

    // Second loop to adjust sizes now that we know more about the panes constraints.
    readjustSizes(leftToAllocate, ungrowable, unshrinkable) {
      let equalSpaceToAllocate;
      if (leftToAllocate > 0) equalSpaceToAllocate = leftToAllocate / (this.panesCount - ungrowable.length);
      else equalSpaceToAllocate = leftToAllocate / (this.panesCount - unshrinkable.length);

      this.panes.forEach((pane, i) => {
        if (leftToAllocate > 0 && !ungrowable.includes(pane.id)) {
          // Need to diff the size before and after to get the exact allocated space.
          const newPaneSize = Math.max(Math.min(pane.size + equalSpaceToAllocate, pane.max), pane.min);
          const allocated = newPaneSize - pane.size;
          leftToAllocate -= allocated;
          pane.size = newPaneSize;
        } else if (!unshrinkable.includes(pane.id)) {
          // Need to diff the size before and after to get the exact allocated space.
          const newPaneSize = Math.max(Math.min(pane.size + equalSpaceToAllocate, pane.max), pane.min);
          const allocated = newPaneSize - pane.size;
          leftToAllocate -= allocated;
          pane.size = newPaneSize;
        }

        // Update each pane through the registered `update` method.
        pane.update({
          [this.horizontal ? "height" : "width"]: `${this.indexedPanes[pane.id].size}%`,
        });
      });

      if (Math.abs(leftToAllocate) > 0.1) {
        // > 0.1: Prevent maths rounding issues due to bytes.
        // Don't emit on hot reload when Vue destroys panes.
        this.$nextTick(() => {
          if (this.ready) {
            // eslint-disable-next-line no-console
            console.warn("Splitpanes: Could not resize panes correctly due to their constraints.");
          }
        });
      }
    },

    /* distributeEmptySpace () {
      let growablePanes = []
      let collapsedPanesCount = 0
      let growableAmount = 0 // Total of how much the current panes can grow to fill blank space.
      let spaceToDistribute = 100 - this.panes.reduce((sum, pane) => (sum += pane.size) && sum, 0)
      // Do a first loop to determine if we can distribute the new blank space between all the
      // expandedPanes, without expanding the collapsed ones.
      this.panes.forEach(pane => {
        if (pane.size < pane.max) growablePanes.push(pane)

        if (!pane.size) collapsedPanesCount++
        else growableAmount += pane.max - pane.size
      })

      // If the blank space to distribute is too great for the expanded panes, also expand collapsed ones.
      let expandCollapsedPanes = growableAmount < spaceToDistribute

      // New space to distribute equally.
      let growablePanesCount = (growablePanes.length - (expandCollapsedPanes ? 0 : collapsedPanesCount))
      let equalSpaceToDistribute = spaceToDistribute / growablePanesCount
      // if (growablePanesCount === 1) equalSpace = 100 / this.panesCount
      let spaceLeftToDistribute = spaceToDistribute

      // Now add the equalSpaceToDistribute to each pane size accordingly.
      growablePanes.forEach(pane => {
        if (pane.size < pane.max && (pane.size || (!pane.size && expandCollapsedPanes))) {
          const newSize = Math.min(pane.size + equalSpaceToDistribute, pane.max)
          let allocatedSpace = (newSize - pane.size)
          spaceLeftToDistribute -= allocatedSpace
          pane.size = newSize
          // If the equalSpaceToDistribute is not fully added to the current pane, distribute the remainder
          // to the next panes.
          // Also fix decimal issue due to bites - E.g. calculating 8.33 and getting 8.3299999999999
          if (equalSpaceToDistribute - allocatedSpace > 0.1) equalSpaceToDistribute = spaceLeftToDistribute / (--growablePanesCount)
        }
      })

      /* Disabled otherwise will show up on hot reload.
      // if there is still space to allocate show warning message.
      if (this.panesCount && ~~spaceLeftToDistribute) {
        // eslint-disable-next-line no-console
        console.warn('Splitpanes: Could not distribute all the empty space between panes due to their constraints.')
      } *\/

      this.$emit('resized', this.panes.map(pane => ({ min: pane.min, max: pane.max, size: pane.size })))
    } */
  },

  watch: {
    panes: {
      // Every time a pane is updated, update the panes accordingly.
      deep: true,
      immediate: false,
      handler() {
        this.updatePaneComponents();
      },
    },
    horizontal() {
      this.updatePaneComponents();
    },
    firstSplitter() {
      this.redoSplitters();
    },
    dblClickSplitter(enable) {
      const splitters = [...this.container.querySelectorAll(".splitpanes__splitter")];
      splitters.forEach((splitter, i) => {
        splitter.ondblclick = enable ? (event) => this.onSplitterDblClick(event, i) : undefined;
      });
    },
  },

  beforeDestroy() {
    // Prevent emitting console warnings on hot reloading.
    this.ready = false;
  },

  mounted() {
    this.container = this.$refs.container;
    this.checkSplitpanesNodes();
    this.redoSplitters();
    this.resetPaneSizes();
    this.$emit("ready");
    this.ready = true;
  },

  render() {
    return h(
      "div",
      {
        ref: "container",
        class: [
          "splitpanes",
          `splitpanes--${this.horizontal ? "horizontal" : "vertical"}`,
          {
            "splitpanes--dragging": this.touch.dragging,
          },
        ],
      },
      this.$slots.default
    );
  },
};
</script>

<style lang="scss">
.splitpanes {
  display: flex;
  width: 100%;
  height: 100%;

  &--vertical {
    flex-direction: row;
  }
  &--horizontal {
    flex-direction: column;
  }
  &--dragging * {
    user-select: none;
  }

  &__pane {
    width: 100%;
    height: 100%;
    overflow: hidden;

    .splitpanes--vertical & {
      transition: width 0.2s ease-out;
    }
    .splitpanes--horizontal & {
      transition: height 0.2s ease-out;
    }
    .splitpanes--dragging & {
      transition: none;
    }
  }

  // Disable default zoom behavior on touch device when double tapping splitter.
  &__splitter {
    touch-action: none;
  }
  &--vertical > .splitpanes__splitter {
    min-width: 1px;
    cursor: col-resize;
  }
  &--horizontal > .splitpanes__splitter {
    min-height: 1px;
    cursor: row-resize;
  }
}
.splitpanes.default-theme {
  .splitpanes__pane {
    background-color: #f2f2f2;
  }
  .splitpanes__splitter {
    background-color: #fff;
    box-sizing: border-box;
    position: relative;
    flex-shrink: 0;
    &:before,
    &:after {
      content: "";
      position: absolute;
      top: 50%;
      left: 50%;
      background-color: rgba(0, 0, 0, 0.15);
      transition: background-color 0.3s;
    }
    &:hover:before,
    &:hover:after {
      background-color: rgba(0, 0, 0, 0.25);
    }
    &:first-child {
      cursor: auto;
    }
  }
}
.default-theme {
  &.splitpanes .splitpanes .splitpanes__splitter {
    z-index: 1;
  }
  &.splitpanes--vertical > .splitpanes__splitter,
  .splitpanes--vertical > .splitpanes__splitter {
    width: 7px;
    border-left: 1px solid #eee;
    margin-left: -1px;
    &:before,
    &:after {
      transform: translateY(-50%);
      width: 1px;
      height: 30px;
    }
    &:before {
      margin-left: -2px;
    }
    &:after {
      margin-left: 1px;
    }
  }
  &.splitpanes--horizontal > .splitpanes__splitter,
  .splitpanes--horizontal > .splitpanes__splitter {
    height: 7px;
    border-top: 1px solid #eee;
    margin-top: -1px;
    &:before,
    &:after {
      transform: translateX(-50%);
      width: 30px;
      height: 1px;
    }
    &:before {
      margin-top: -2px;
    }
    &:after {
      margin-top: 1px;
    }
  }
}
</style>
