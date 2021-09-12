<ol class="breadcrumb" aria-label="breadcrumbs">
    <li data-bs-toggle="tooltip" data-bs-placement="top" title="所属标签" class="breadcrumb-item"><a href="/tags/{{$data->tag->id}}.html">
            <img class="tag-icon" src="{{$data->tag->icon}}" alt="{{$data->tag->name}}">
            {{$data->tag->name}}
        </a>
    </li>
    <li data-bs-toggle="tooltip" data-bs-placement="top" title="浏览量" class="breadcrumb-item active"><a href="#">
            <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-eye" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                <circle cx="12" cy="12" r="2"></circle>
                <path d="M22 12c-2.667 4.667 -6 7 -10 7s-7.333 -2.333 -10 -7c2.667 -4.667 6 -7 10 -7s7.333 2.333 10 7"></path>
            </svg>
            {{$data->view}}
        </a>
    </li>
    <li data-bs-toggle="tooltip" data-bs-placement="top" title="{{$data->created_at}}" class="breadcrumb-item active" aria-current="page"><a  href="#">
            发布于:{{format_date($data->created_at)}}
        </a>
    </li>
    <li data-bs-toggle="tooltip" data-bs-placement="top" title="{{$data->updated_at}}" class="breadcrumb-item active" aria-current="page"><a href="#">
            更新于:{{format_date($data->updated_at)}}
        </a>
    </li>
    @if(auth()->check())

        @if(Authority()->check("admin_topic_edit") && curd()->GetUserClass(auth()->data()->class_id)['permission-value']>curd()->GetUserClass($data->user->class_id)['permission-value'])
            <li data-bs-toggle="tooltip" data-bs-placement="top" title="修改帖子信息" class="breadcrumb-item">
                <a href="/topic/{{$data->id}}/edit">
                    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-pencil-square" viewBox="0 0 16 16">
                        <path d="M15.502 1.94a.5.5 0 0 1 0 .706L14.459 3.69l-2-2L13.502.646a.5.5 0 0 1 .707 0l1.293 1.293zm-1.75 2.456-2-2L4.939 9.21a.5.5 0 0 0-.121.196l-.805 2.414a.25.25 0 0 0 .316.316l2.414-.805a.5.5 0 0 0 .196-.12l6.813-6.814z"/>
                        <path fill-rule="evenodd" d="M1 13.5A1.5 1.5 0 0 0 2.5 15h11a1.5 1.5 0 0 0 1.5-1.5v-6a.5.5 0 0 0-1 0v6a.5.5 0 0 1-.5.5h-11a.5.5 0 0 1-.5-.5v-11a.5.5 0 0 1 .5-.5H9a.5.5 0 0 0 0-1H2.5A1.5 1.5 0 0 0 1 2.5v11z"/>
                    </svg>
                    编辑
                </a>
            </li>
        @elseif(Authority()->check("topic_edit") && auth()->id() === $data->user->id)
            <li data-bs-toggle="tooltip" data-bs-placement="top" title="修改帖子信息" class="breadcrumb-item">
                <a href="/topic/{{$data->id}}/edit">
                    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-pencil-square" viewBox="0 0 16 16">
                        <path d="M15.502 1.94a.5.5 0 0 1 0 .706L14.459 3.69l-2-2L13.502.646a.5.5 0 0 1 .707 0l1.293 1.293zm-1.75 2.456-2-2L4.939 9.21a.5.5 0 0 0-.121.196l-.805 2.414a.25.25 0 0 0 .316.316l2.414-.805a.5.5 0 0 0 .196-.12l6.813-6.814z"/>
                        <path fill-rule="evenodd" d="M1 13.5A1.5 1.5 0 0 0 2.5 15h11a1.5 1.5 0 0 0 1.5-1.5v-6a.5.5 0 0 0-1 0v6a.5.5 0 0 1-.5.5h-11a.5.5 0 0 1-.5-.5v-11a.5.5 0 0 1 .5-.5H9a.5.5 0 0 0 0-1H2.5A1.5 1.5 0 0 0 1 2.5v11z"/>
                    </svg>
                    编辑
                </a>
            </li>
        @endif


    @endif


</ol>