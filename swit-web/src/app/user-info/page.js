'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';

export default function UserInfo() {
  const [user, setUser] = useState(null);
  const [error, setError] = useState('');
  const router = useRouter();

  useEffect(() => {
    const fetchUserInfo = async () => {
      const accessToken = localStorage.getItem('access_token');
      const username = localStorage.getItem('username');
      console.log('Access token received:', accessToken);
      console.log('Username received:', username);

      if (!accessToken || !username) {
        console.log('Access token or username does not exist, redirecting to login page');
        router.push('/login');
        return;
      }

      try {
        const response = await fetch(`http://localhost:9000/users/username/${username}`, {
          headers: {
            'Authorization': `Bearer ${accessToken}`,
          },
        });

        console.log('User information request status:', response.status);

        if (response.ok) {
          const data = await response.json();
          console.log('User information received:', data);
          setUser(data);
        } else if (response.status === 401) {
          console.log('Unauthorized, clearing local storage and redirecting to login page');
          localStorage.removeItem('access_token');
          localStorage.removeItem('refresh_token');
          localStorage.removeItem('username');
          router.push('/login');
        } else {
          setError('Failed to get user information');
        }
      } catch (error) {
        console.error('Error occurred while fetching user information:', error);
        setError('An error occurred, please try again later');
      }
    };

    fetchUserInfo();
  }, [router]);

  const handleLogout = async () => {
    try {
      const accessToken = localStorage.getItem('access_token');
      const response = await fetch('http://localhost:9001/auth/logout', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${accessToken}`,
        },
      });

      if (response.ok) {
        localStorage.removeItem('access_token');
        localStorage.removeItem('refresh_token');
        localStorage.removeItem('username');
        router.push('/login');
      } else {
        setError('Logout failed, please try again later');
      }
    } catch (error) {
      setError('An error occurred, please try again later');
    }
  };

  if (error) {
    return <div className="text-red-500">{error}</div>;
  }

  if (!user) {
    return <div>Loading...</div>;
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 py-12 px-4 sm:px-6 lg:px-8">
      <div className="max-w-md w-full space-y-8">
        <div>
          <h2 className="mt-6 text-center text-3xl font-extrabold text-gray-900">User Information</h2>
        </div>
        <div className="mt-8 space-y-6">
          <div className="bg-white shadow overflow-hidden sm:rounded-lg p-6">
            <p className="text-lg text-gray-800"><strong className="font-medium text-gray-900">Username:</strong> {user.username}</p>
            <p className="text-lg text-gray-800 mt-4"><strong className="font-medium text-gray-900">Email:</strong> {user.email}</p>
          </div>
          <div>
            <button
              onClick={handleLogout}
              className="group relative w-full flex justify-center py-2 px-4 border border-transparent text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
            >
              Logout
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
