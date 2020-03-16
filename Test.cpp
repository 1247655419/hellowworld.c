#include<string>
#include<iostream>
#include <process.h>
#include <atlstr.h>
#include"Finder.h"
using namespace std;
//线程函数 
UINT FinderEntry(LPVOID lpParam)
{
    //CRapidFinder通过参数传递进来  
    CMyFinder* pFinder = (CMyFinder*)lpParam;
    CDirectoryNode* pNode = NULL;
    BOOL bActive = TRUE; //bActive为TRUE,表示当前线程激活 
    string pathname;
    //循环处理m_listDir列表中的目录 
    while (1)
    {
        //从列表中取出一个目录 
        EnterCriticalSection(&pFinder->m_Section);
        if (pFinder->m_listDir.IsEmpty()) //目录列表为空，当前线程不激活，所以bAactive=FALSE 
        {
            bActive = FALSE;
        }
        else
        {
            pNode = pFinder->m_listDir.GetHead(); //得到一个目录 
            pFinder->m_listDir.Remove(pNode);    //从目录列表中移除 
            pathname = pNode->szDir;

        }
        LeaveCriticalSection(&pFinder->m_Section);
        //如果停止当前线程 
        if (bActive == FALSE)
        {
            //停止当前线程 
            EnterCriticalSection(&pFinder->m_Section);
            pFinder->m_nThreadCount--;

            //如果当前活动线程数为0,跳出，结束 
            if (pFinder->m_nThreadCount == 0)
            {
                LeaveCriticalSection(&pFinder->m_Section);
                break;
            }
            LeaveCriticalSection(&pFinder->m_Section);
            //当前活动线程数不为0，等待其他线程向目录列表中加目录 
            ResetEvent(pFinder->m_hDirEvent);
            WaitForSingleObject(pFinder->m_hDirEvent, INFINITE);

            //运行到这，就说明其他线程向目录列表中加入了新的目录 
            EnterCriticalSection(&pFinder->m_Section);
            pFinder->m_nThreadCount++; //激活了自己的线程，线程数++ 
            LeaveCriticalSection(&pFinder->m_Section);
            bActive = TRUE; //目录不再为空 
            continue; //跳到while,重新在目录列表中取目录 
        }
        //从目录列表中成功取得了目录 
        WIN32_FIND_DATA fileinfo;
        HANDLE hFindFile;
        //生成正确的查找字符串 
        if (pNode->szDir[strlen(pNode->szDir) - 1] != '\\')
        {
            strcat_s(pNode->szDir, "\\");
        }
        strcat_s(pNode->szDir, "*.*");
        //查找文件的框架
        CString temp = pNode->szDir;
        hFindFile = FindFirstFile((LPWSTR)(LPCTSTR)temp, &fileinfo);
        if (hFindFile != INVALID_HANDLE_VALUE)
        {
            do
            {
                //如果是当前目录，跳过 
                if (fileinfo.cFileName[0] == '.')
                {
                    continue;
                }
                //如果是目录 
                if (fileinfo.dwFileAttributes & FILE_ATTRIBUTE_DIRECTORY)
                {
                    //将当前目录加入到目录列表 
                    CDirectoryNode* p = new CDirectoryNode;
                    strncpy_s(p->szDir, pNode->szDir, strlen(pNode->szDir) - 3); //将pNode后面的*.*三位去掉 
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

                    //使一个线程从非活动状态变成活动状态 
                    SetEvent(pFinder->m_hDirEvent);
                }
                else //如果是文件 
                {
                    //判断是否为要查找的文件  
                    CString tem = fileinfo.cFileName;
                    int nInStrLen = wcslen(tem);
                    int nOutStrLen = WideCharToMultiByte(CP_ACP, 0, tem, nInStrLen, NULL, 0, 0, 0) + 2;
                    char* nOutStr = new char[nOutStrLen];
                    memset(nOutStr, 0x00, nOutStrLen);
                    WideCharToMultiByte(CP_ACP, 0, tem, nInStrLen, nOutStr, nOutStrLen, 0, 0);
                    if (pFinder->CheckFile(nOutStr)) //符合查找的文件  
                    {
                        //打印 
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

    //促使一个搜索线程从WaitForSingleObject返回，并退出循环 
    SetEvent(pFinder->m_hDirEvent);

    //判断此线程是否是最后一个结束循环的线程，如果是就通知主线程 
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
    //开始开启多线程 
    for (int i = 0; i < pFinder->m_nMaxThread; i++)
    {
        CreateThread(NULL, 0, (LPTHREAD_START_ROUTINE)FinderEntry, pFinder, 0, NULL);
    }

    //只有m_hExitEvent受信状态，主线程才恢复运行 
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