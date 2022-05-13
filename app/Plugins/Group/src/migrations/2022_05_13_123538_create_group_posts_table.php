<?php

use Hyperf\Database\Schema\Schema;
use Hyperf\Database\Schema\Blueprint;
use Hyperf\Database\Migrations\Migration;

class CreateGroupPostsTable extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::create('group_posts', function (Blueprint $table) {
            $table->bigIncrements('id');
			$table->string('user_id');
			$table->string('markdown');
			$table->string('content');
			$table->string('user_ua');
			$table->string('user_ip');
            $table->timestamps();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('group_posts');
    }
}
