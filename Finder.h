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
    CMyFinder(int nMaxThread); //���캯�� 
    virtual ~CMyFinder();    //�������� 
    BOOL    CheckFile(char* lpszFileName); //���lpszFileName�Ƿ���ϲ������� 
    int     m_nResultCount; //�ҵ��Ľ������ 
    int     m_nThreadCount; //��ǰ���߳����� 
    CTypedMyList<CDirectoryNode*> m_listDir; //����Ŀ¼ 
    CRITICAL_SECTION    m_Section;   //���� 
    const int   m_nMaxThread;   //����߳����� 
    char    m_szMatchName[MAX_PATH]; //Ҫ���ҵ����� 
    HANDLE  m_hDirEvent;    //�����Ŀ¼����λ 
    HANDLE  m_hExitEvent;   //�����߳��˳�ʱ��λ 
};