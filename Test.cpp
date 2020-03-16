#include<string>
#include<iostream>
#include <process.h>
#include <atlstr.h>
#include"Finder.h"
using namespace std;
//�̺߳��� 
UINT FinderEntry(LPVOID lpParam)
{
    //CRapidFinderͨ���������ݽ���  
    CMyFinder* pFinder = (CMyFinder*)lpParam;
    CDirectoryNode* pNode = NULL;
    BOOL bActive = TRUE; //bActiveΪTRUE,��ʾ��ǰ�̼߳��� 
    string pathname;
    //ѭ������m_listDir�б��е�Ŀ¼ 
    while (1)
    {
        //���б���ȡ��һ��Ŀ¼ 
        EnterCriticalSection(&pFinder->m_Section);
        if (pFinder->m_listDir.IsEmpty()) //Ŀ¼�б�Ϊ�գ���ǰ�̲߳��������bAactive=FALSE 
        {
            bActive = FALSE;
        }
        else
        {
            pNode = pFinder->m_listDir.GetHead(); //�õ�һ��Ŀ¼ 
            pFinder->m_listDir.Remove(pNode);    //��Ŀ¼�б����Ƴ� 
            pathname = pNode->szDir;

        }
        LeaveCriticalSection(&pFinder->m_Section);
        //���ֹͣ��ǰ�߳� 
        if (bActive == FALSE)
        {
            //ֹͣ��ǰ�߳� 
            EnterCriticalSection(&pFinder->m_Section);
            pFinder->m_nThreadCount--;

            //�����ǰ��߳���Ϊ0,���������� 
            if (pFinder->m_nThreadCount == 0)
            {
                LeaveCriticalSection(&pFinder->m_Section);
                break;
            }
            LeaveCriticalSection(&pFinder->m_Section);
            //��ǰ��߳�����Ϊ0���ȴ������߳���Ŀ¼�б��м�Ŀ¼ 
            ResetEvent(pFinder->m_hDirEvent);
            WaitForSingleObject(pFinder->m_hDirEvent, INFINITE);

            //���е��⣬��˵�������߳���Ŀ¼�б��м������µ�Ŀ¼ 
            EnterCriticalSection(&pFinder->m_Section);
            pFinder->m_nThreadCount++; //�������Լ����̣߳��߳���++ 
            LeaveCriticalSection(&pFinder->m_Section);
            bActive = TRUE; //Ŀ¼����Ϊ�� 
            continue; //����while,������Ŀ¼�б���ȡĿ¼ 
        }
        //��Ŀ¼�б��гɹ�ȡ����Ŀ¼ 
        WIN32_FIND_DATA fileinfo;
        HANDLE hFindFile;
        //������ȷ�Ĳ����ַ��� 
        if (pNode->szDir[strlen(pNode->szDir) - 1] != '\\')
        {
            strcat_s(pNode->szDir, "\\");
        }
        strcat_s(pNode->szDir, "*.*");
        //�����ļ��Ŀ��
        CString temp = pNode->szDir;
        hFindFile = FindFirstFile((LPWSTR)(LPCTSTR)temp, &fileinfo);
        if (hFindFile != INVALID_HANDLE_VALUE)
        {
            do
            {
                //����ǵ�ǰĿ¼������ 
                if (fileinfo.cFileName[0] == '.')
                {
                    continue;
                }
                //�����Ŀ¼ 
                if (fileinfo.dwFileAttributes & FILE_ATTRIBUTE_DIRECTORY)
                {
                    //����ǰĿ¼���뵽Ŀ¼�б� 
                    CDirectoryNode* p = new CDirectoryNode;
                    strncpy_s(p->szDir, pNode->szDir, strlen(pNode->szDir) - 3); //��pNode�����*.*��λȥ�� 
                    CString tem = fileinfo.cFileName;
                    int nInStrLen = wcslen(tem);
                    int nOutStrLen = WideCharToMultiByte(CP_ACP, 0, tem, nInStrLen, NULL, 0, 0, 0) + 2;
                    char* nOutStr = new char[nOutStrLen];
                    memset(nOutStr, 0x00, nOutStrLen);
                    WideCharToMultiByte(CP_ACP, 0, tem, nInStrLen, nOutStr, nOutStrLen, 0, 0);
                    strcat_s(p->szDir, (const char*)(nOutStr));
                    delete(nOutStr);
                    nOutStr = NULL;
                    EnterCriticalSection(&pFinder->m_Section);
                    pFinder->m_listDir.AddHead(p);
                    LeaveCriticalSection(&pFinder->m_Section);

                    //ʹһ���̴߳ӷǻ״̬��ɻ״̬ 
                    SetEvent(pFinder->m_hDirEvent);
                }
                else //������ļ� 
                {
                    //�ж��Ƿ�ΪҪ���ҵ��ļ�  
                    CString tem = fileinfo.cFileName;
                    int nInStrLen = wcslen(tem);
                    int nOutStrLen = WideCharToMultiByte(CP_ACP, 0, tem, nInStrLen, NULL, 0, 0, 0) + 2;
                    char* nOutStr = new char[nOutStrLen];
                    memset(nOutStr, 0x00, nOutStrLen);
                    WideCharToMultiByte(CP_ACP, 0, tem, nInStrLen, nOutStr, nOutStrLen, 0, 0);
                    if (pFinder->CheckFile(nOutStr)) //���ϲ��ҵ��ļ�  
                    {
                        //��ӡ 
                        EnterCriticalSection(&pFinder->m_Section);
                        pFinder->m_nResultCount++;
                        LeaveCriticalSection(&pFinder->m_Section);
                        cout << "count:"<<pFinder->m_nResultCount<<endl;
                        cout << "path:  "<<pathname + nOutStr << endl;
                    }
                    delete(nOutStr);
                    nOutStr = NULL;
                }
            } while (FindNextFile(hFindFile, &fileinfo));
        }
    }

    //��ʹһ�������̴߳�WaitForSingleObject���أ����˳�ѭ�� 
    SetEvent(pFinder->m_hDirEvent);

    //�жϴ��߳��Ƿ������һ������ѭ�����̣߳�����Ǿ�֪ͨ���߳� 
    if (WaitForSingleObject(pFinder->m_hDirEvent, 0) != WAIT_TIMEOUT)
    {
        SetEvent(pFinder->m_hExitEvent);
    }
    return 1;
}
void TestFindFile(std::string Path, std::string FileName) {
    CMyFinder* pFinder = new CMyFinder(64);
    CDirectoryNode* pNode = new CDirectoryNode;

    strcpy_s(pNode->szDir, Path.c_str());
    pFinder->m_listDir.AddHead(pNode);

    strcpy_s(pFinder->m_szMatchName, FileName.c_str());
    pFinder->m_nThreadCount = pFinder->m_nMaxThread;
    //��ʼ�������߳� 
    for (int i = 0; i < pFinder->m_nMaxThread; i++)
    {
        CreateThread(NULL, 0, (LPTHREAD_START_ROUTINE)FinderEntry, pFinder, 0, NULL);
    }

    //ֻ��m_hExitEvent����״̬�����̲߳Żָ����� 
    WaitForSingleObject(pFinder->m_hExitEvent, INFINITE);
    cout << "total file:" << pFinder->m_nResultCount << endl;
    if (pFinder != NULL) {
        delete pFinder;
        pFinder = NULL;
    }
    return;
}
int main() {
	string path = "d:\\temp";
	string filename= "22.txt";
	cout << "input search path:";
	//cin >> path;
	cout << "input filename:";
	//cin >> filename;
    TestFindFile(path, filename);
	return 0;
}