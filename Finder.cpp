
#include "finder.h"

//m_nMaxThread ��const int���ͣ�ֻ��ͨ�����ַ�ʽ��ʼ�� 
CMyFinder::CMyFinder(int nMaxThread) :m_nMaxThread(nMaxThread)
{
    m_nResultCount = 0;
    m_nThreadCount = 0;
    m_listDir.Construct(offsetof(CDirectoryNode, pNext));  //offsetof��stddef.hͷ�ļ��� 
    ::InitializeCriticalSection(&m_Section);
    m_szMatchName[0] = '\0';
    m_hDirEvent = ::CreateEvent(NULL, FALSE, FALSE, NULL);
    m_hExitEvent = ::CreateEvent(NULL, FALSE, FALSE, NULL);

}

CMyFinder::~CMyFinder()
{
    ::DeleteCriticalSection(&m_Section);
    ::CloseHandle(m_hDirEvent);
    ::CloseHandle(m_hExitEvent);
}

BOOL CMyFinder::CheckFile(char* lpszFileName)
{
    //���������ַ��� 

    char string[MAX_PATH];
    char strSearch[MAX_PATH];
    strcpy_s(string, lpszFileName);
    strcpy_s(strSearch, m_szMatchName);

    //���ַ�����д 
    _strupr_s(string);
    _strupr_s(strSearch);

    //�Ƚ�string���Ƿ���strSearch 
    if (strstr(string, strSearch) != NULL)
    {
        return TRUE;
    }
    return FALSE;
}

