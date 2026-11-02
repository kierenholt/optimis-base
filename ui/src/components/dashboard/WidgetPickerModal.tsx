import React, { useEffect, useMemo, useState } from 'react';
import { MonochromeLucideIcon } from '../MonochromeLucideIcon';
import {
  getWidgetCompositeKey,
  PlacedDashboardWidget,
  PluginWidgetDescriptor,
} from '../../themeAndTypes';
import { WidgetFootprintDiagram } from './WidgetFootprintDiagram';
import { WidgetPickerGalleryView } from './WidgetPickerGalleryView';
import { WidgetPickerSplitView } from './WidgetPickerSplitView';

export interface WidgetPickerModalProps {
  isOpen: boolean;
  availableWidgets: PluginWidgetDescriptor[];
  placedWidgets: PlacedDashboardWidget[];
  onClose: () => void;
  onSelectWidget: (widget: PluginWidgetDescriptor, e: React.MouseEvent<HTMLButtonElement>) => void;
  onRegisterWidgetIframeRef: (widgetId: string, pluginId: string, el: HTMLIFrameElement | null) => void;
}

export const WidgetPickerModal: React.FC<WidgetPickerModalProps> = ({
  isOpen,
  availableWidgets,
  placedWidgets,
  onClose,
  onSelectWidget,
  onRegisterWidgetIframeRef,
}) => {
  const [sizeFilter, setSizeFilter] = useState<'all' | '1x1' | '2x1' | '1x2' | '2x2'>('all');
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [pluginFilter, setPluginFilter] = useState<string>('all');
  const [hidePlaced, setHidePlaced] = useState<boolean>(false);
  const [modalViewMode, setModalViewMode] = useState<'gallery' | 'split'>('gallery');
  const [selectedSplitWidgetKey, setSelectedSplitWidgetKey] = useState<string>('');

  useEffect(() => {
    if (isOpen) {
      setSizeFilter('all');
      setSearchQuery('');
      setPluginFilter('all');
      setHidePlaced(false);
    }
  }, [isOpen]);

  const pickerPlugins = useMemo(() => {
    const map = new Map<string, string>();
    for (const w of availableWidgets) {
      if (!map.has(w.plugin_id)) {
        map.set(w.plugin_id, w.plugin_title || w.plugin_id);
      }
    }
    return Array.from(map.entries()).map(([id, title]) => ({ id, title }));
  }, [availableWidgets]);

  const filteredPickerWidgets = useMemo(() => {
    const q = searchQuery.trim().toLowerCase();
    return availableWidgets.filter((w) => {
      const sizeKey = `${w.width}x${w.height}`;
      if (sizeFilter !== 'all' && sizeKey !== sizeFilter) {
        return false;
      }
      if (pluginFilter !== 'all' && w.plugin_id !== pluginFilter) {
        return false;
      }
      if (hidePlaced) {
        const isPlaced = placedWidgets.some((pw) => pw.plugin_id === w.plugin_id && pw.id === w.id);
        if (isPlaced) {
          return false;
        }
      }
      if (q !== '') {
        const matchesTitle = (w.title || '').toLowerCase().includes(q);
        const matchesDesc = (w.description || '').toLowerCase().includes(q);
        const matchesPlugin = (w.plugin_title || w.plugin_id || '').toLowerCase().includes(q);
        const matchesId = (w.id || '').toLowerCase().includes(q);
        const matchesKw = Array.isArray(w.keywords)
          ? w.keywords.some((kw) => kw.toLowerCase().includes(q))
          : false;
        if (!matchesTitle && !matchesDesc && !matchesPlugin && !matchesId && !matchesKw) {
          return false;
        }
      }
      return true;
    });
  }, [availableWidgets, sizeFilter, pluginFilter, hidePlaced, searchQuery, placedWidgets]);

  const selectedSplitWidget = useMemo(() => {
    if (filteredPickerWidgets.length === 0) {
      return null;
    }
    if (selectedSplitWidgetKey) {
      const found = filteredPickerWidgets.find(
        (w) => getWidgetCompositeKey(w.plugin_id, w.id) === selectedSplitWidgetKey
      );
      if (found) {
        return found;
      }
    }
    return filteredPickerWidgets[0];
  }, [filteredPickerWidgets, selectedSplitWidgetKey]);

  if (!isOpen) {
    return null;
  }

  return (
    <div
      data-testid="dashboard-widget-picker-backdrop"
      onClick={onClose}
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 1000,
        backgroundColor: 'rgba(15, 23, 42, 0.55)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        padding: '20px',
        boxSizing: 'border-box',
      }}
    >
      <div
        data-testid="dashboard-widget-picker-modal"
        data-modal-view-mode={modalViewMode}
        role="dialog"
        aria-modal="true"
        aria-label="Add Dashboard Widget"
        onClick={(e) => e.stopPropagation()}
        style={{
          width: '94vw',
          maxWidth: '1240px',
          height: '86vh',
          maxHeight: '880px',
          backgroundColor: 'var(--mis-bg-surface)',
          border: '1px solid var(--mis-border-strong)',
          borderRadius: 'var(--mis-radius-lg, 12px)',
          boxShadow: '0 20px 50px rgba(0, 0, 0, 0.35)',
          display: 'flex',
          flexDirection: 'column',
          overflow: 'hidden',
        }}
      >
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            gap: '12px',
            padding: '14px 18px',
            borderBottom: '1px solid var(--mis-border-subtle)',
            backgroundColor: 'var(--mis-bg-elevated)',
            flexWrap: 'wrap',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <MonochromeLucideIcon name="layout-grid" size={16} />
            <span style={{ fontSize: 'calc(14px * var(--mis-font-scale, 1))', fontWeight: 700 }}>
              Select a Widget to Place on Grid
            </span>
            <span
              data-testid="widget-picker-result-count"
              style={{
                fontSize: 'calc(11px * var(--mis-font-scale, 1))',
                fontWeight: 600,
                color: 'var(--mis-text-secondary)',
                padding: '2px 8px',
                borderRadius: '999px',
                border: '1px solid var(--mis-border-subtle)',
                backgroundColor: 'var(--mis-bg-surface)',
              }}
            >
              {filteredPickerWidgets.length} of {availableWidgets.length} widgets
            </span>
          </div>

          <div style={{ display: 'inline-flex', alignItems: 'center', gap: '10px' }}>
            <div
              data-testid="widget-picker-view-mode-toggle"
              role="group"
              aria-label="Widget picker view mode"
              style={{
                display: 'inline-flex',
                alignItems: 'center',
                padding: '2px',
                borderRadius: 'var(--mis-radius-sm, 4px)',
                border: '1px solid var(--mis-border-subtle)',
                backgroundColor: 'var(--mis-bg-surface)',
                gap: '2px',
              }}
            >
              <button
                type="button"
                data-testid="widget-picker-view-mode-gallery"
                aria-pressed={modalViewMode === 'gallery'}
                onClick={() => setModalViewMode('gallery')}
                style={{
                  padding: '5px 10px',
                  borderRadius: 'var(--mis-radius-sm, 4px)',
                  border: 'none',
                  backgroundColor:
                    modalViewMode === 'gallery' ? 'var(--mis-brand-primary)' : 'transparent',
                  color:
                    modalViewMode === 'gallery'
                      ? 'var(--mis-brand-primary-fg)'
                      : 'var(--mis-text-primary)',
                  fontSize: 'calc(11px * var(--mis-font-scale, 1))',
                  fontWeight: 700,
                  cursor: 'pointer',
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: '5px',
                }}
              >
                <MonochromeLucideIcon name="layout-grid" size={13} />
                <span>Gallery View</span>
              </button>
              <button
                type="button"
                data-testid="widget-picker-view-mode-split"
                aria-pressed={modalViewMode === 'split'}
                onClick={() => setModalViewMode('split')}
                style={{
                  padding: '5px 10px',
                  borderRadius: 'var(--mis-radius-sm, 4px)',
                  border: 'none',
                  backgroundColor:
                    modalViewMode === 'split' ? 'var(--mis-brand-primary)' : 'transparent',
                  color:
                    modalViewMode === 'split'
                      ? 'var(--mis-brand-primary-fg)'
                      : 'var(--mis-text-primary)',
                  fontSize: 'calc(11px * var(--mis-font-scale, 1))',
                  fontWeight: 700,
                  cursor: 'pointer',
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: '5px',
                }}
              >
                <MonochromeLucideIcon name="columns" size={13} />
                <span>Split Preview</span>
              </button>
            </div>

            <button
              type="button"
              data-testid="dashboard-widget-picker-close-btn"
              aria-label="Close widget picker modal"
              onClick={onClose}
              style={{
                background: 'none',
                border: '1px solid var(--mis-border-subtle)',
                borderRadius: 'var(--mis-radius-sm, 4px)',
                color: 'var(--mis-text-secondary)',
                cursor: 'pointer',
                padding: '4px 6px',
                display: 'inline-flex',
                alignItems: 'center',
              }}
            >
              <MonochromeLucideIcon name="x" size={14} />
            </button>
          </div>
        </div>

        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            gap: '10px',
            padding: '12px 18px',
            borderBottom: '1px solid var(--mis-border-subtle)',
            backgroundColor: 'var(--mis-bg-surface)',
          }}
        >
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              gap: '10px',
              flexWrap: 'wrap',
            }}
          >
            <div
              role="group"
              aria-label="Filter widgets by footprint size"
              style={{ display: 'inline-flex', alignItems: 'center', gap: '6px', flexWrap: 'wrap' }}
            >
              <button
                type="button"
                data-testid="widget-picker-size-filter-all"
                aria-pressed={sizeFilter === 'all'}
                onClick={() => setSizeFilter('all')}
                style={{
                  padding: '6px 12px',
                  borderRadius: 'var(--mis-radius-sm, 4px)',
                  border:
                    sizeFilter === 'all'
                      ? '2px solid var(--mis-brand-primary)'
                      : '1px solid var(--mis-border-subtle)',
                  backgroundColor:
                    sizeFilter === 'all'
                      ? 'var(--mis-brand-primary)'
                      : 'var(--mis-bg-elevated)',
                  color:
                    sizeFilter === 'all'
                      ? 'var(--mis-brand-primary-fg)'
                      : 'var(--mis-text-primary)',
                  fontSize: 'calc(11px * var(--mis-font-scale, 1))',
                  fontWeight: 700,
                  cursor: 'pointer',
                }}
              >
                All Sizes
              </button>
              {(
                [
                  { id: '1x1', w: 1, h: 1 },
                  { id: '2x1', w: 2, h: 1 },
                  { id: '1x2', w: 1, h: 2 },
                  { id: '2x2', w: 2, h: 2 },
                ] as const
              ).map((opt) => {
                const active = sizeFilter === opt.id;
                return (
                  <button
                    key={opt.id}
                    type="button"
                    data-testid={`widget-picker-size-filter-${opt.id}`}
                    aria-label={`Filter ${opt.w} by ${opt.h} footprint`}
                    aria-pressed={active}
                    onClick={() => setSizeFilter(opt.id)}
                    style={{
                      padding: '3px',
                      borderRadius: 'var(--mis-radius-sm, 4px)',
                      border: active
                        ? '2px solid var(--mis-brand-primary)'
                        : '1px solid var(--mis-border-subtle)',
                      backgroundColor: active
                        ? 'var(--mis-bg-elevated)'
                        : 'transparent',
                      cursor: 'pointer',
                      display: 'inline-flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                    }}
                  >
                    <WidgetFootprintDiagram
                      testId={`size-filter-footprint-diagram-${opt.id}`}
                      width={opt.w}
                      height={opt.h}
                      cellWidthPx={14}
                      cellHeightPx={10}
                    />
                  </button>
                );
              })}
            </div>

            <div style={{ display: 'inline-flex', alignItems: 'center', gap: '8px', flexWrap: 'wrap' }}>
              <select
                data-testid="widget-picker-plugin-filter"
                aria-label="Filter widgets by plugin"
                value={pluginFilter}
                onChange={(e) => setPluginFilter(e.target.value)}
                style={{
                  padding: '5px 8px',
                  borderRadius: 'var(--mis-radius-sm, 4px)',
                  border: '1px solid var(--mis-border-subtle)',
                  backgroundColor: 'var(--mis-bg-elevated)',
                  color: 'var(--mis-text-primary)',
                  fontSize: 'calc(12px * var(--mis-font-scale, 1))',
                  fontWeight: 600,
                }}
              >
                <option value="all">All Plugins</option>
                {pickerPlugins.map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.title}
                  </option>
                ))}
              </select>

              <button
                type="button"
                data-testid="widget-picker-hide-placed-toggle"
                aria-pressed={hidePlaced}
                onClick={() => setHidePlaced((prev) => !prev)}
                style={{
                  padding: '5px 10px',
                  borderRadius: 'var(--mis-radius-sm, 4px)',
                  border: hidePlaced
                    ? '1px solid var(--mis-brand-primary)'
                    : '1px solid var(--mis-border-subtle)',
                  backgroundColor: hidePlaced
                    ? 'var(--mis-brand-primary)'
                    : 'var(--mis-bg-elevated)',
                  color: hidePlaced
                    ? 'var(--mis-brand-primary-fg)'
                    : 'var(--mis-text-primary)',
                  fontSize: 'calc(11px * var(--mis-font-scale, 1))',
                  fontWeight: 700,
                  cursor: 'pointer',
                }}
              >
                Hide already placed
              </button>
            </div>
          </div>

          <div style={{ position: 'relative', display: 'flex', alignItems: 'center' }}>
            <input
              type="search"
              data-testid="widget-picker-search-input"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search widgets by title, description, plugin, or keyword..."
              style={{
                width: '100%',
                padding: '7px 10px',
                borderRadius: 'var(--mis-radius-sm, 4px)',
                border: '1px solid var(--mis-border-subtle)',
                backgroundColor: 'var(--mis-bg-elevated)',
                color: 'var(--mis-text-primary)',
                fontSize: 'calc(12px * var(--mis-font-scale, 1))',
                boxSizing: 'border-box',
              }}
            />
          </div>
        </div>

        {modalViewMode === 'gallery' ? (
          <WidgetPickerGalleryView
            widgets={filteredPickerWidgets}
            placedWidgets={placedWidgets}
            onSelectWidget={onSelectWidget}
          />
        ) : (
          <WidgetPickerSplitView
            widgets={filteredPickerWidgets}
            placedWidgets={placedWidgets}
            selectedWidget={selectedSplitWidget}
            onSelectWidgetKey={setSelectedSplitWidgetKey}
            onSelectWidget={onSelectWidget}
            onRegisterWidgetIframeRef={onRegisterWidgetIframeRef}
          />
        )}
      </div>
    </div>
  );
};
