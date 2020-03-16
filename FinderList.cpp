#include"FinderList.h"
void MyList::AddHead(void* p)
{
    *GetNextPtr(p) = m_pHead;
    m_pHead = p;
}

BOOL MyList::Remove(void* p)
{
    if (p == NULL)
    {
        return FALSE;
    }

    BOOL bResult = FALSE;
    if (p == m_pHead)
    {
        m_pHead = *GetNextPtr(m_pHead);
        bResult = TRUE;
    }
    else
    {
        void* pTest = m_pHead;
        while (pTest != NULL && *GetNextPtr(pTest) != p)
        {
            pTest = *GetNextPtr(pTest);
        }
        if (pTest != NULL)
        {
            *GetNextPtr(pTest) = *GetNextPtr(p);
            bResult = TRUE;
        }
    }

    return bResult;
}

void* CNoTrackObject::operator new(size_t nSize)
{
    void* p = GlobalAlloc(GPTR, nSize);
    return  p;
}

void CNoTrackObject::operator delete(void* p)
{
    if (p != NULL)
    {
        GlobalFree(p);
    }
}