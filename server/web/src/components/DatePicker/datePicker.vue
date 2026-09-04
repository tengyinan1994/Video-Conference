<template>
  <n-date-picker
    v-bind="$props"
    v-model:value="modelValue"
    :shortcuts="showShortcuts ? shortcuts : undefined"
    :clearable="true"
    :on-update:show="handleShow"
    style="width: 100%"
  />
</template>

<script lang="ts">
  import { computed, defineComponent, onBeforeUnmount, onMounted, ref } from 'vue';
  import {
    dateToTimestamp,
    defRangeShortcuts,
    defShortcuts,
    formatToDate,
    formatToDateTime,
    timestampToTime,
  } from '@/utils/dateUtil';
  import { basicProps } from './props';

  const VIEWPORT_MARGIN = 8;
  const PANEL_MIN_HEIGHT = 160;

  export default defineComponent({
    name: 'DatePicker',
    props: {
      ...basicProps,
    },
    emits: ['update:formValue', 'update:startValue', 'update:endValue'],
    setup(props, { emit }) {
      const shortcuts = ref<any>({});
      let fittedPanel: HTMLElement | null = null;
      let fitTimer: number | undefined;

      function invokeShowCallback(show: boolean) {
        const cb = (props as any).onUpdateShow ?? (props as any)['onUpdate:show'];
        if (typeof cb === 'function') {
          cb(show);
        } else if (Array.isArray(cb)) {
          cb.forEach((fn) => fn?.(show));
        }
      }

      function resetDatePanelFit() {
        if (fitTimer !== undefined) {
          window.clearTimeout(fitTimer);
          fitTimer = undefined;
        }
        window.removeEventListener('resize', scheduleFit);
        if (!fittedPanel) {
          return;
        }
        fittedPanel.style.maxHeight = '';
        fittedPanel.style.overflowY = '';
        fittedPanel.style.marginTop = '';
        fittedPanel = null;
      }

      function fitDatePanelToViewport() {
        const panel = document.querySelector('.n-date-panel.n-date-panel--shadow') as HTMLElement | null;
        if (!panel) {
          return;
        }
        fittedPanel = panel;
        const vh = window.innerHeight;
        panel.style.marginTop = '';
        panel.style.maxHeight = `${Math.max(vh - VIEWPORT_MARGIN * 2, PANEL_MIN_HEIGHT)}px`;
        panel.style.overflowY = 'auto';

        const rect = panel.getBoundingClientRect();
        if (rect.top < VIEWPORT_MARGIN) {
          panel.style.marginTop = `${Math.round(VIEWPORT_MARGIN - rect.top)}px`;
        }

        const shifted = panel.getBoundingClientRect();
        if (shifted.bottom > vh - VIEWPORT_MARGIN) {
          panel.style.maxHeight = `${Math.max(
            Math.floor(vh - VIEWPORT_MARGIN - shifted.top),
            PANEL_MIN_HEIGHT
          )}px`;
        }
      }

      function scheduleFit() {
        requestAnimationFrame(() => {
          requestAnimationFrame(fitDatePanelToViewport);
        });
      }

      function handleShow(show: boolean) {
        invokeShowCallback(show);
        if (show) {
          window.addEventListener('resize', scheduleFit);
          scheduleFit();
          fitTimer = window.setTimeout(fitDatePanelToViewport, 200);
        } else {
          resetDatePanelFit();
        }
      }

      onBeforeUnmount(() => {
        resetDatePanelFit();
      });

      function getTimestamp(value) {
        let t = dateToTimestamp(value);
        console.log('getTimestamp t:' + t);
        if (t === 0) {
          return undefined;
        }
        return t;
      }

      function setTimestamp(value) {
        if (value === undefined) {
          return undefined;
        }
        if (!isTimeType()) {
          return formatToDate(new Date(Number(value)).toDateString());
        } else {
          return formatToDateTime(timestampToTime(Number(value / 1000)));
        }
      }

      function isRangeType() {
        return props.type.indexOf('range') != -1;
      }

      function isTimeType() {
        return props.type.indexOf('time') != -1;
      }

      const modelValue = computed({
        get() {
          if (!isRangeType()) {
            return getTimestamp(props.formValue);
          } else {
            const value = [getTimestamp(props.startValue), getTimestamp(props.endValue)];
            if (!value[0] && !value[1]) {
              return null;
            }
            return value;
          }
        },
        set(value) {
          if (!isRangeType()) {
            emit('update:formValue', setTimestamp(value));
          } else {
            emit('update:startValue', setTimestamp(value[0]));
            emit('update:endValue', setTimestamp(value[1]));
          }
        },
      });

      onMounted(async () => {
        if (!isRangeType()) {
          shortcuts.value = defShortcuts();
        } else {
          shortcuts.value = defRangeShortcuts();
        }
      });

      return {
        modelValue,
        shortcuts,
        showShortcuts: props.showShortcuts,
        handleShow,
      };
    },
  });
</script>

<style lang="less">
  .n-date-panel--shadow {
    max-height: calc(100vh - 16px);
    overflow-y: auto;
    overscroll-behavior: contain;
  }

  .n-date-panel--shadow .n-date-panel-header {
    position: sticky;
    top: 0;
    z-index: 4;
    background-color: var(--n-panel-color);
  }

  .n-date-panel--shadow .n-date-panel-actions {
    position: sticky;
    bottom: 0;
    z-index: 4;
    background-color: var(--n-panel-color);
  }
</style>
