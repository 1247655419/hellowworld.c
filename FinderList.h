#include <Windows.h> 

class MyList
{
public:
    MyList(int nNextOffset = 0);
    void Construct(int nNextOffset);
    BOOL IsEmpty() const;
    void AddHead(void* p);
    void RemoveAll();
    void* GetHead() const;
    void* GetNext(void* p) const;
    BOOL Remove(void* p);

    //为实现接口所需要的成员 
    void* m_pHead;
    int m_nNextOffset;
    void** GetNextPtr(void* p) const;
};

//类的内联函数 
inline MyList::MyList(int nNextOffset)
{
    m_pHead = NULL; 
    m_nNextOffset = nNextOffset;
}

inline void MyList::Construct(int nNextOffset)
{
    m_nNextOffset = nNextOffset;
}

inline BOOL MyList::IsEmpty() const
{
    return m_pHead == NULL;
}

inline void MyList::RemoveAll()
{
    m_pHead = NULL;
}

inline void* MyList::GetHead() const
{
    return m_pHead;
}

inline void* MyList::GetNext(void* preElement) const
{
    return *GetNextPtr(preElement);
}

inline void** MyList::GetNextPtr(void* p) const
{
    return (void**)((BYTE*)p + m_nNextOffset);
}

class CNoTrackObject
{
public:
    void* operator new(size_t nSize);
    void operator delete(void*);
    virtual ~CNoTrackObject() {};
};

template<class TYPE>

class CTypedMyList :public MyList
{
public:
    CTypedMyList(int nNextOffset = 0)
        :MyList(nNextOffset) {}
    void AddHead(TYPE p)
    {
        MyList::AddHead((void*)p);
    }

    TYPE GetHead()
    {
        return (TYPE)MyList::GetHead();
    }

    TYPE GetNext(TYPE p)
    {
        return (TYPE)MyList::GetNext((void*)p);
    }

    BOOL Remove(TYPE p)
    {
        return MyList::Remove(p);
    }

    operator TYPE()
    {
        return (TYPE)MyList::GetHead();
    }
};