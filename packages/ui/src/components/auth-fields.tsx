'use client';

import React from 'react';
import { Input, type InputProps } from './input';
import { PasswordInput, type PasswordInputProps } from './password-input';
import { OTPInput, type OTPInputProps } from './otp-input';

export interface AuthFieldProps extends InputProps {}

export const AuthField: React.FC<AuthFieldProps> = (props) => {
  return <Input {...props} />;
};

export interface PhoneFieldProps extends InputProps {}

export const PhoneField: React.FC<PhoneFieldProps> = ({ placeholder = '+234 800 000 0000', ...props }) => {
  return <Input type="tel" placeholder={placeholder} inputMode="tel" autoComplete="tel" {...props} />;
};

export interface PasswordFieldProps extends PasswordInputProps {}

export const PasswordField: React.FC<PasswordFieldProps> = (props) => {
  return <PasswordInput {...props} />;
};

export interface OTPFieldProps extends OTPInputProps {}

export const OTPField: React.FC<OTPFieldProps> = (props) => {
  return <OTPInput {...props} />;
};
