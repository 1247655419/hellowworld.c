
#include "finder.h"

//m_nMaxThread 是const int类型，只能通过这种方式初始化 
CMyFinder::CMyFinder(int nMaxThread) :m_nMaxThread(nMaxThread)
{
    m_nResultCount = 0;
    m_nThreadCount = 0;
    m_listDir.Construct(offsetof(CDirectoryNode, pNext));  //offsetof在stddef.h头文件中 
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
    //定义两个字符串 

    char string[MAX_PATH];
    char strSearch[MAX_PATH];
    strcpy_s(string, lpszFileName);
    strcpy_s(strSearch, m_szMatchName);

    //将字符串大写 
    _strupr_s(string);
    _strupr_s(strSearch);

    //比较string中是否含有strSearch 
    if (strstr(string, strSearch) != NULL)
    {
        return TRUE;
    }
    return FALSE;
}

