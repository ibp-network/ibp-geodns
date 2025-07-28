import React from 'react';
import './DateRangePicker.css';

const DateRangePicker = ({ dateRange, onChange }) => {
  return (
    <div className="date-controls">
      <input
        type="date"
        value={dateRange.start.toISOString().split('T')[0]}
        onChange={(e) => onChange({ ...dateRange, start: new Date(e.target.value) })}
        className="date-input"
      />
      <span className="date-separator">to</span>
      <input
        type="date"
        value={dateRange.end.toISOString().split('T')[0]}
        onChange={(e) => onChange({ ...dateRange, end: new Date(e.target.value) })}
        className="date-input"
      />
    </div>
  );
};

export default DateRangePicker;