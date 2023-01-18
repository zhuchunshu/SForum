<?php

declare(strict_types=1);

use Hyperf\Database\Seeders\Seeder;

class TopicTagSeeder extends Seeder
{
    /**
     * Run the database seeds.
     *
     * @return void
     */
    public function run()
    {
        \Hyperf\DbConnection\Db::table('topic_tag')->insert([
           'name' => '默认板块',
           'color' => '#000000',
            'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-message" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M4 21v-13a3 3 0 0 1 3 -3h10a3 3 0 0 1 3 3v6a3 3 0 0 1 -3 3h-9l-4 4"></path>
   <line x1="8" y1="9" x2="16" y2="9"></line>
   <line x1="8" y1="13" x2="14" y2="13"></line>
</svg>',
            'description' => '默认板块描述',
            'type' => null,
            'created_at' => date("Y-m-d H:i:s"),
            'updated_at' => date("Y-m-d H:i:s"),
            'font_color' => '#FFFFFF'
        ]);
    }
}
