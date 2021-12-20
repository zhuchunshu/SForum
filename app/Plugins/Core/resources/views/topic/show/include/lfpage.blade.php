<div class="col-md-12">
    <div class="border-0 card">
        <div class="card-body">
            <ul class="pagination ">

                @if ($get_topic['shang'])
                    <li class="page-item page-prev">
                        <a class="page-link"
                           href="/{{$get_topic['shang']['id']}}.html">
                            <div class="page-item-subtitle">上一篇文章</div>
                            <div class="page-item-title">
                                {{ \Hyperf\Utils\Str::limit($get_topic['shang']['title'], 20, '...') }}</div>
                        </a>
                    </li>
                @else
                    <li class="page-item page-prev disabled">
                        <a class="page-link" href="#" tabindex="-1" aria-disabled="true">
                            <div class="page-item-subtitle">上一篇文章</div>
                            <div class="page-item-title">暂无</div>
                        </a>
                    </li>
                @endif
                @if ($get_topic['xia'])
                    <li class="page-item page-next">
                        <a class="page-link"
                           href="/{{ $get_topic['xia']['id']  }}.html">
                            <div class="page-item-subtitle">下一篇文章</div>
                            <div class="page-item-title">
                                {{ \Hyperf\Utils\Str::limit($get_topic['xia']['title'], 20, '...') }}</div>
                        </a>
                    </li>
                @else
                    <li class="page-item page-next disabled">
                        <a class="page-link" href="#">
                            <div class="page-item-subtitle">下一篇文章</div>
                            <div class="page-item-title">暂无</div>
                        </a>
                    </li>
                @endif
            </ul>
        </div>
    </div>
</div>