#include "FinderList.h"
#include <Windows.h> 
struct CDirectoryNode : public CNoTrackObject
{
    CDirectoryNode* pNext;
    char szDir[MAX_PATH];
};

class CMyFinder
{
public:
    CMyFinder(int nMaxThread); //构造函数 
    virtual ~CMyFinder();    //析构函数 
    BOOL    CheckFile(char* lpszFileName); //检查lpszFileName是否符合查找条件 
    int     m_nResultCount; //找到的结果数量 
    int     m_nThreadCount; //当前的线程数量 
    CTypedMyList<CDirectoryNode*> m_listDir; //查找目录 
    CRITICAL_SECTION    m_Section;   //共享 
    const int   m_nMaxThread;   //最大线程数量 
    char    m_szMatchName[MAX_PATH]; //要查找的名称 
    HANDLE  m_hDirEvent;    //添加新目录后置位 
    HANDLE  m_hExitEvent;   //所有线程退出时置位 
};