A golang implementation for one billion rows challenge.

### Data Prep
Data is generated using create_meansurements.py file. It creates a file of around 14.5GB

### Implementation

- The initial implementation was reading the file line by line and storing intermediate results in hashmap. And finally computing the stats for each city from the hashmap.
    - Total time taken for this was: 1 min 53 sec
- Some intermediate solutions I tried included having a having a go routine which would process the file and push each line to a channel. This channel would then be processed by another go routine.
    - In this approach through cpu profiling found that the write to channel was getting blocked as the other go routine(processing from channel) was slow.
- Then referred to some blogs where processing a file by chunk was mentioned.
    - In this approach first based on file stats the required chunks and their offsets are figured out. Then we spawn a go routine for each chunk which does processing on it.
    - Finally all these go routines push data -> a processed hashmap out of that chunk to a shared channel
    - This channel is then processed by the main thread/ go routine thus avoiding the use of locks.
    - Using this approach with 10 chunks the following time was observed: `159.82s user 21.63s system 572% cpu 31.677 total`
    - Note that **~160sec** user time means all goroutines processing times.
    - But total time was only **32 sec**.

### Code Run and Profiling Commands

#### To run the code:

`go run main.go`

#### To generate profile

Add following code:
```
f1, err := os.Create("cpuProfile.prof")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer f1.Close()
	pprof.StartCPUProfile(f1)
	defer pprof.StopCPUProfile()
```
This creates a cpuProfile.prof file

#### Generate image to analyse prof data
`go tool pprof -png cpuProfile.prof`

This geenrates a png image. Add few in the repo for reference