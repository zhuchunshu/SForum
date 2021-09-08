<ol class="breadcrumb" aria-label="breadcrumbs">
    <li class="breadcrumb-item"><a href="/tags/{{$data->tag->id}}.html">
            <img class="tag-icon" src="{{$data->tag->icon}}" alt="{{$data->tag->name}}">
            {{$data->tag->name}}
        </a></li>
    <li class="breadcrumb-item active"><a href="#">
            <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-eye" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                <circle cx="12" cy="12" r="2"></circle>
                <path d="M22 12c-2.667 4.667 -6 7 -10 7s-7.333 -2.333 -10 -7c2.667 -4.667 6 -7 10 -7s7.333 2.333 10 7"></path>
            </svg>
            {{$data->view}}
        </a>
    </li>
    <li class="breadcrumb-item active" aria-current="page"><a href="#">
            发布于:{{format_date($data->created_at)}}
        </a>
    </li>
    <li class="breadcrumb-item active" aria-current="page"><a href="#">
            更新于:{{format_date($data->created_at)}}
        </a>
    </li>

</ol>