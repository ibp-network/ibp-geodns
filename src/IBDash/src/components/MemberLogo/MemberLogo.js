import React from 'react';
import './MemberLogo.css';

const MemberLogo = ({ member, size = 'medium', className = '' }) => {
  const sizeClasses = {
    small: 'logo-small',
    medium: 'logo-medium',
    large: 'logo-large'
  };

  const initials = member.name.substring(0, 2).toUpperCase();

  return (
    <>
      {member.logo ? (
        <img
          src={member.logo}
          alt={member.name}
          className={`member-logo ${sizeClasses[size]} ${className}`}
          onError={(e) => {
            e.target.style.display = 'none';
            e.target.nextSibling.style.display = 'flex';
          }}
        />
      ) : null}
      <div
        className={`logo-placeholder ${sizeClasses[size]} ${className}`}
        style={{ display: member.logo ? 'none' : 'flex' }}
      >
        {initials}
      </div>
    </>
  );
};

export default MemberLogo;