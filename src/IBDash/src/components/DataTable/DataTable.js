import React, { useState } from 'react';
import './DataTable.css';

const DataTable = ({ data, type }) => {
  const [sortField, setSortField] = useState('requests');
  const [sortOrder, setSortOrder] = useState('desc');
  const [currentPage, setCurrentPage] = useState(1);
  const itemsPerPage = 10;

  const handleSort = (field) => {
    if (field === sortField) {
      setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc');
    } else {
      setSortField(field);
      setSortOrder('desc');
    }
  };

  const sortedData = [...data].sort((a, b) => {
    const aVal = a[sortField] || 0;
    const bVal = b[sortField] || 0;
    
    if (typeof aVal === 'string') {
      return sortOrder === 'asc' ? aVal.localeCompare(bVal) : bVal.localeCompare(aVal);
    }
    return sortOrder === 'asc' ? aVal - bVal : bVal - aVal;
  });

  const paginatedData = sortedData.slice(
    (currentPage - 1) * itemsPerPage,
    currentPage * itemsPerPage
  );

  const totalPages = Math.ceil(sortedData.length / itemsPerPage);

  const getColumns = () => {
    switch (type) {
      case 'country':
        return [
          { key: 'date', label: 'Date', sortable: true },
          { key: 'country', label: 'Country Code', sortable: true },
          { key: 'country_name', label: 'Country Name', sortable: true },
          { key: 'requests', label: 'Requests', sortable: true }
        ];
      case 'asn':
        return [
          { key: 'date', label: 'Date', sortable: true },
          { key: 'asn', label: 'ASN', sortable: true },
          { key: 'network', label: 'Network', sortable: true },
          { key: 'requests', label: 'Requests', sortable: true }
        ];
      case 'service':
        return [
          { key: 'date', label: 'Date', sortable: true },
          { key: 'service', label: 'Service', sortable: true },
          { key: 'domain', label: 'Domain', sortable: true },
          { key: 'requests', label: 'Requests', sortable: true }
        ];
      case 'member':
        return [
          { key: 'date', label: 'Date', sortable: true },
          { key: 'member', label: 'Member', sortable: true },
          { key: 'requests', label: 'Requests', sortable: true }
        ];
      default:
        return [];
    }
  };

  const columns = getColumns();

  return (
    <div className="data-table-container">
      <div className="table-wrapper">
        <table className="data-table">
          <thead>
            <tr>
              {columns.map(col => (
                <th
                  key={col.key}
                  className={col.sortable ? 'sortable' : ''}
                  onClick={() => col.sortable && handleSort(col.key)}
                >
                  <div className="th-content">
                    {col.label}
                    {col.sortable && sortField === col.key && (
                      <span className="sort-icon">
                        {sortOrder === 'asc' ? '↑' : '↓'}
                      </span>
                    )}
                  </div>
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {paginatedData.map((row, index) => (
              <tr key={index} className="fade-in">
                {columns.map(col => (
                  <td key={col.key}>
                    {col.key === 'requests' ? (
                      <span className="requests-value">
                        {row[col.key]?.toLocaleString()}
                      </span>
                    ) : (
                      row[col.key] || '-'
                    )}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {totalPages > 1 && (
        <div className="table-pagination">
          <button
            className="pagination-btn"
            onClick={() => setCurrentPage(1)}
            disabled={currentPage === 1}
          >
            First
          </button>
          <button
            className="pagination-btn"
            onClick={() => setCurrentPage(currentPage - 1)}
            disabled={currentPage === 1}
          >
            Previous
          </button>
          <span className="pagination-info">
            Page {currentPage} of {totalPages}
          </span>
          <button
            className="pagination-btn"
            onClick={() => setCurrentPage(currentPage + 1)}
            disabled={currentPage === totalPages}
          >
            Next
          </button>
          <button
            className="pagination-btn"
            onClick={() => setCurrentPage(totalPages)}
            disabled={currentPage === totalPages}
          >
            Last
          </button>
        </div>
      )}
    </div>
  );
};

export default DataTable;