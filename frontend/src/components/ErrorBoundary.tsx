import { Component, ReactNode } from 'react';

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

export default class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo): void {
    console.error('ErrorBoundary caught an error:', error, errorInfo);
  }

  handleReset = (): void => {
    this.setState({ hasError: false, error: null });
  };

  render(): ReactNode {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback;
      }

      return (
        <div className="min-h-[200px] flex items-center justify-center">
          <div className="text-center p-8 bg-danger-50 rounded-lg border border-danger-200 max-w-md">
            <svg
              className="w-12 h-12 text-danger-500 mx-auto mb-4"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
              />
            </svg>
            <h3 className="text-lg font-semibold text-danger-800 mb-2">
              오류가 발생했습니다
            </h3>
            <p className="text-sm text-danger-600 mb-4">
              {this.state.error?.message || '알 수 없는 오류가 발생했습니다.'}
            </p>
            <button
              onClick={this.handleReset}
              className="btn-danger"
            >
              다시 시도
            </button>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}
